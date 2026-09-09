// PicoClaw - Ultra-lightweight personal AI agent

package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/constants"
	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"

	"github.com/sipeed/picoclaw/pkg/config"
)

// CallLLM performs an LLM call with fallback support, hook invocation, and retry logic.
// It handles PreLLM setup, the actual LLM invocation with retry, and AfterLLM processing.
// Returns Control indicating what the coordinator should do next.
func (p *Pipeline) CallLLM(
	ctx context.Context,
	turnCtx context.Context,
	ts *turnState,
	exec *turnExecution,
	iteration int,
) (Control, error) {
	al := p.al
	maxMediaSize := p.Cfg.Agents.Defaults.GetMaxMediaSize()

	// PreLLM: resolve media refs (except on iteration 1 where user media is already resolved)
	if iteration > 1 {
		exec.messages = resolveMediaRefs(exec.messages, p.MediaStore, maxMediaSize, exec.currentTurnStart)
	}

	// PreLLM: graceful terminal handling
	exec.gracefulTerminal, _ = ts.gracefulInterruptRequested()
	exec.providerToolDefs = ts.agent.Tools.ToProviderDefs()
	exec.providerToolDefs = filterToolsByTurnProfile(exec.providerToolDefs, ts.profile)

	// Native web search support
	webSearchEnabled := al.cfg.Tools.IsToolEnabled("web") && turnProfileToolAllowed(ts.profile, "web_search")
	exec.useNativeSearch = webSearchEnabled && al.cfg.Tools.Web.PreferNative &&
		func() bool {
			if ns, ok := ts.agent.Provider.(providers.NativeSearchCapable); ok {
				return ns.SupportsNativeSearch()
			}
			return false
		}()
	if exec.useNativeSearch {
		filtered := make([]providers.ToolDefinition, 0, len(exec.providerToolDefs))
		for _, td := range exec.providerToolDefs {
			if td.Function.Name != "web_search" {
				filtered = append(filtered, td)
			}
		}
		exec.providerToolDefs = filtered
	}

	exec.callMessages = exec.messages
	if exec.gracefulTerminal {
		exec.callMessages = append(append([]providers.Message(nil), exec.messages...), ts.interruptHintMessage())
		exec.providerToolDefs = nil
		ts.markGracefulTerminalUsed()
	}
	if err := p.routeMediaTurn(ts, exec); err != nil {
		return ControlBreak, err
	}

	exec.llmOpts = map[string]any{
		"max_tokens":       ts.agent.MaxTokens,
		"temperature":      ts.agent.Temperature,
		"prompt_cache_key": ts.agent.ID,
	}
	if exec.useNativeSearch {
		exec.llmOpts["native_search"] = true
	}
	applyTurnThinkingOptions(exec, ts.agent, exec.activeProvider, true)

	exec.llmModel = exec.activeModel

	// BeforeLLM hook
	if p.Hooks != nil {
		llmReq, decision := p.Hooks.BeforeLLM(turnCtx, &LLMHookRequest{
			Meta:             ts.eventMeta("runTurn", "turn.llm.request"),
			Context:          cloneTurnContext(ts.turnCtx),
			Model:            exec.llmModel,
			Messages:         exec.callMessages,
			Tools:            exec.providerToolDefs,
			Options:          exec.llmOpts,
			GracefulTerminal: exec.gracefulTerminal,
		})
		switch decision.normalizedAction() {
		case HookActionContinue, HookActionModify:
			if llmReq != nil {
				prevModel := exec.llmModel
				exec.llmModel = llmReq.Model
				exec.callMessages = llmReq.Messages
				exec.providerToolDefs = filterToolsByTurnProfile(llmReq.Tools, ts.profile)
				exec.llmOpts = llmReq.Options
				nativeSearchAllowed := exec.useNativeSearch &&
					turnProfileToolAllowed(ts.profile, "web_search")
				if !nativeSearchAllowed {
					delete(exec.llmOpts, "native_search")
				}
				if strings.TrimSpace(exec.llmModel) != "" && exec.llmModel != prevModel {
					p.applyBeforeLLMModelRewrite(ts, exec)
					applyTurnThinkingOptions(exec, ts.agent, exec.activeProvider, true)
				}
			}
		case HookActionAbortTurn:
			cancelConfiguredStreamingLLM(turnCtx, exec)
			exec.abortedByHook = true
			return ControlBreak, nil
		case HookActionHardAbort:
			cancelConfiguredStreamingLLM(turnCtx, exec)
			_ = ts.requestHardAbort()
			exec.abortedByHardAbort = true
			return ControlBreak, nil
		}
	}

	al.emitEvent(
		runtimeevents.KindAgentLLMRequest,
		ts.eventMeta("runTurn", "turn.llm.request"),
		LLMRequestPayload{
			Model:         exec.llmModel,
			MessagesCount: len(exec.callMessages),
			ToolsCount:    len(exec.providerToolDefs),
			MaxTokens:     ts.agent.MaxTokens,
			Temperature:   ts.agent.Temperature,
		},
	)

	logger.DebugCF("agent", "LLM request",
		map[string]any{
			"agent_id":          ts.agent.ID,
			"iteration":         iteration,
			"model":             exec.llmModel,
			"messages_count":    len(exec.callMessages),
			"tools_count":       len(exec.providerToolDefs),
			"max_tokens":        ts.agent.MaxTokens,
			"temperature":       ts.agent.Temperature,
			"system_prompt_len": len(exec.callMessages[0].Content),
		})
	// LLM call closure with fallback support
	callLLM := func(messagesForCall []providers.Message, toolDefsForCall []providers.ToolDefinition) (*providers.LLMResponse, error) {
		providerCtx, providerCancel := context.WithCancel(turnCtx)
		ts.setProviderCancel(providerCancel)
		defer func() {
			providerCancel()
			ts.clearProviderCancel(providerCancel)
		}()

		al.activeRequestsInc()
		defer al.activeRequestsDec()

		if response, handled, streamErr := p.tryConfiguredStreamingLLM(
			providerCtx,
			ts,
			exec,
			messagesForCall,
			toolDefsForCall,
		); handled {
			return response, streamErr
		}

		runCandidate := func(
			ctx context.Context,
			candidate providers.FallbackCandidate,
		) (*providers.LLMResponse, error) {
			candidateProvider, err := providerForFallbackCandidate(
				ts.agent,
				exec.activeProvider,
				exec.activeCandidates,
				candidate,
			)
			if err != nil {
				return nil, err
			}
			callOpts := shallowCloneLLMOptions(exec.llmOpts)
			delete(callOpts, "thinking_level")
			candidateCfg := resolveActiveModelConfig(
				p.Cfg,
				ts.agent.Workspace,
				[]providers.FallbackCandidate{candidate},
				candidate.Model,
				p.Cfg.Agents.Defaults.Provider,
			)
			// activeThinkingSettings, not the model config alone: the direct
			// single-candidate path falls back to the agent's thinking level
			// when the candidate has no configured one, and the first candidate
			// must not be sent different generation parameters just because a
			// fallback exists.
			candidateThinking := activeThinkingSettings(ts.agent, candidateCfg)
			applyThinkingOption(callOpts, candidateProvider, candidateThinking, true, ts.agent.ID)
			exec.suppressReasoning = shouldSuppressReasoningFor(candidateThinking)
			// Each attempt gets its own copy of the conversation. A provider
			// adapter that rewrote the slice it was handed would otherwise
			// corrupt what the next candidate sees, and the media attached to
			// the turn is exactly the content that would be lost.
			return candidateProvider.Chat(
				ctx, cloneMessagesForAttempt(messagesForCall), toolDefsForCall, candidate.Model, callOpts,
			)
		}

		if len(exec.activeCandidates) > 1 && p.Fallback != nil {
			var (
				fbResult *providers.FallbackResult
				fbErr    error
			)
			if hasMediaRefs(messagesForCall) {
				// ExecuteImageCandidate, not ExecuteImage: recovering the
				// candidate by matching provider and model cannot tell two
				// model_list entries for the same model apart, and would hand
				// both attempts the same configuration.
				fbResult, fbErr = p.Fallback.ExecuteImageCandidate(
					providerCtx,
					exec.activeCandidates,
					runCandidate,
				)
			} else {
				fbResult, fbErr = p.Fallback.WithToolTurn(len(toolDefsForCall) > 0).ExecuteCandidate(
					providerCtx,
					exec.activeCandidates,
					runCandidate,
				)
			}
			if fbErr != nil {
				return nil, fbErr
			}
			if fbResult.Provider != "" && len(fbResult.Attempts) > 0 {
				selected := providerAttemptPayload(
					ts,
					providers.FallbackCandidate{
						Provider:    fbResult.Provider,
						Model:       fbResult.Model,
						IdentityKey: fbResult.IdentityKey,
					},
					fbResult.Model,
					len(fbResult.Attempts)+1,
					len(fbResult.Attempts),
				)
				al.emitProviderEvent(
					runtimeevents.KindProviderFallbackSelected,
					ts.eventMeta("runTurn", "provider.fallback"),
					selected,
				)
				logger.InfoCF(
					"agent",
					fmt.Sprintf("Fallback: succeeded with %s/%s after %d attempts",
						fbResult.Provider, fbResult.Model, len(fbResult.Attempts)+1),
					map[string]any{"agent_id": ts.agent.ID, "iteration": iteration},
				)
			}
			for _, candidate := range exec.activeCandidates {
				if candidate.StableKey() != fbResult.IdentityKey {
					continue
				}
				exec.llmModelName = resolvedCandidateModelName(
					[]providers.FallbackCandidate{candidate},
					exec.llmModelName,
				)
				break
			}
			return fbResult.Response, nil
		}
		return exec.activeProvider.Chat(providerCtx, messagesForCall, toolDefsForCall, exec.llmModel, exec.llmOpts)
	}

	// Retry loop
	var err error
	maxRetries := p.Cfg.Agents.Defaults.MaxLLMRetries
	if maxRetries <= 0 {
		maxRetries = 2
	}
	backoffSecs := p.Cfg.Agents.Defaults.LLMRetryBackoffSecs
	if backoffSecs <= 0 {
		backoffSecs = 2
	}
	// When a fallback candidate exists, trying a different provider beats
	// waiting out a backoff on the one that just failed. Same-candidate retries
	// are therefore capped at one, so the worst case stays bounded rather than
	// multiplying retries by candidates by agent iterations.
	hasFallbackCandidates := len(exec.activeCandidates) > 1 && p.Fallback != nil
	if hasFallbackCandidates && maxRetries > 1 {
		maxRetries = 1
	}

	for retry := 0; retry <= maxRetries; retry++ {
		traceTurnLifecycle("provider_started", ts, map[string]any{
			"attempt": retry + 1,
		})
		attemptPayload := p.providerAttemptFor(ts, exec, retry+1)
		al.emitProviderEvent(
			runtimeevents.KindProviderAttemptStarted,
			ts.eventMeta("runTurn", "provider.attempt"),
			attemptPayload,
		)
		attemptStarted := time.Now()

		exec.response, err = callLLM(exec.callMessages, exec.providerToolDefs)
		if err == nil {
			completed := attemptPayload
			completed.DurationMS = time.Since(attemptStarted).Milliseconds()
			al.emitProviderEvent(
				runtimeevents.KindProviderAttemptCompleted,
				ts.eventMeta("runTurn", "provider.attempt"),
				completed,
			)
			traceTurnLifecycle("provider_completed", ts, map[string]any{
				"attempt": retry + 1,
			})
			if responseIsUserVisiblyEmpty(exec.response) {
				// Observability only. A parsed HTTP success that carries no
				// text is not evidence of an outage, and must not trigger a
				// retry or a failover: the model is entitled to say nothing.
				// Genuinely provider-attributable emptiness — a zero-byte body,
				// truncated framing, unparseable JSON — arrives as an error
				// from the provider adapter and is classified there.
				traceTurnLifecycle("provider_empty", ts, map[string]any{
					"attempt": retry + 1,
				})
			}
			break
		}
		if ts.hardAbortRequested() && errors.Is(err, context.Canceled) {
			_ = ts.requestHardAbort()
			exec.abortedByHardAbort = true
			return ControlBreak, nil
		}
		if isConfiguredStreamingVisibleError(err) {
			break
		}

		if hasMediaRefs(exec.callMessages) && isVisionUnsupportedError(err) {
			return ControlBreak, visionUnsupportedModelError(
				exec.llmModelName,
				len(ts.agent.ImageCandidates) > 0,
			)
		}

		// Record the failure against the shared cooldown state.
		//
		// The fallback chain already does this, but it only runs with more than
		// one candidate. A sole configured provider previously failed here with
		// no memory at all, so every turn re-ran the same doomed request.
		failErr := p.classifyAndRecordProviderFailure(ts, exec, err, hasFallbackCandidates)

		failedPayload := attemptPayload
		failedPayload.DurationMS = time.Since(attemptStarted).Milliseconds()
		if failErr != nil {
			failedPayload.ErrorClass = string(failErr.Reason)
			failedPayload.HTTPStatus = failErr.Status
		} else {
			failedPayload.ErrorClass = string(providers.FailoverUnknown)
		}
		al.emitProviderEvent(
			runtimeevents.KindProviderAttemptFailed,
			ts.eventMeta("runTurn", "provider.attempt"),
			failedPayload,
		)

		errMsg := strings.ToLower(err.Error())
		retryReason, isTransientError := transientLLMRetryReason(err)
		isContextError := !isTransientError && (strings.Contains(errMsg, "context_length_exceeded") ||
			strings.Contains(errMsg, "context window") ||
			strings.Contains(errMsg, "context_window") ||
			strings.Contains(errMsg, "maximum context length") ||
			strings.Contains(errMsg, "token limit") ||
			strings.Contains(errMsg, "too many tokens") ||
			strings.Contains(errMsg, "max_tokens") ||
			strings.Contains(errMsg, "invalidparameter") ||
			strings.Contains(errMsg, "prompt is too long") ||
			strings.Contains(errMsg, "request too large"))

		// Auth, billing, hard quota and malformed requests will fail
		// identically a second later, so a same-candidate retry is pure
		// latency. Context overflow is excluded deliberately: it classifies as
		// a request error but has its own compact-and-retry recovery below,
		// which must keep working.
		if failErr != nil && !failErr.AllowsSameCandidateRetry() && !isContextError {
			logger.WarnCF("agent", "Provider failure is not retriable on the same model", map[string]any{
				"agent_id":    ts.agent.ID,
				"error_class": string(failErr.Reason),
				"status":      failErr.Status,
			})
			break
		}

		if isTransientError && retry < maxRetries {
			backoff := time.Duration(retry+1) * time.Duration(backoffSecs) * time.Second
			// A provider that sent Retry-After knows better than our guess, but
			// only up to a point: an unbounded sleep is indistinguishable from a
			// hang to the user, and the backoff stays cancellable either way.
			if failErr != nil && failErr.RetryAfter > backoff {
				backoff = failErr.RetryAfter
				if backoff > maxHonouredRetryAfter {
					backoff = maxHonouredRetryAfter
				}
			}
			al.emitEvent(
				runtimeevents.KindAgentLLMRetry,
				ts.eventMeta("runTurn", "turn.llm.retry"),
				LLMRetryPayload{
					Attempt:    retry + 1,
					MaxRetries: maxRetries,
					Reason:     retryReason,
					Error:      err.Error(),
					Backoff:    backoff,
				},
			)
			scheduled := attemptPayload
			scheduled.ErrorClass = retryReason
			scheduled.RetryDelay = backoff.Milliseconds()
			al.emitProviderEvent(
				runtimeevents.KindProviderRetryScheduled,
				ts.eventMeta("runTurn", "provider.retry"),
				scheduled,
			)
			logger.WarnCF("agent", "Transient LLM error, retrying after backoff", map[string]any{
				"error":   err.Error(),
				"reason":  retryReason,
				"retry":   retry,
				"backoff": backoff.String(),
			})
			if sleepErr := sleepWithContext(turnCtx, backoff); sleepErr != nil {
				if ts.hardAbortRequested() {
					_ = ts.requestHardAbort()
					return ControlBreak, nil
				}
				err = sleepErr
				break
			}
			continue
		}

		if isContextError && retry < maxRetries && !ts.opts.NoHistory {
			al.emitEvent(
				runtimeevents.KindAgentLLMRetry,
				ts.eventMeta("runTurn", "turn.llm.retry"),
				LLMRetryPayload{
					Attempt:    retry + 1,
					MaxRetries: maxRetries,
					Reason:     "context_limit",
					Error:      err.Error(),
				},
			)
			logger.WarnCF(
				"agent",
				"Context window error detected, attempting compression",
				map[string]any{
					"error": err.Error(),
					"retry": retry,
				},
			)

			if retry == 0 && !constants.IsInternalChannel(ts.channel) {
				al.bus.PublishOutbound(ctx, outboundMessageForTurn(
					ts,
					"Context window exceeded. Compressing history and retrying...",
				))
			}

			if compactErr := p.ContextManager.Compact(ctx, &CompactRequest{
				SessionKey: ts.sessionKey,
				Reason:     ContextCompressReasonRetry,
				Budget:     ts.agent.ContextWindow,
			}); compactErr != nil {
				logger.WarnCF("agent", "Context overflow compact failed", map[string]any{
					"session_key": ts.sessionKey,
					"error":       compactErr.Error(),
				})
			}
			ts.refreshRestorePointFromSession(ts.agent)
			if asmResp, asmErr := p.ContextManager.Assemble(ctx, &AssembleRequest{
				SessionKey: ts.sessionKey,
				Budget:     ts.agent.ContextWindow,
				MaxTokens:  ts.agent.MaxTokens,
			}); asmErr == nil && asmResp != nil {
				exec.history = asmResp.History
				exec.summary = asmResp.Summary
			}
			contextualSkills := ts.activeSkills
			if ts.agent.ContextBuilder != nil {
				contextualSkills = ts.agent.ContextBuilder.ResolveActiveSkillsForContext(ts.activeSkills)
			}
			ts.recordSkillContextSnapshot(skillContextTriggerContextRetryRebuild, contextualSkills)
			stableHistory, protectedTurnTail := splitHistoryForActiveTurn(
				exec.history,
				ts.persistedMessagesSnapshot(),
			)
			buildMessages := func(trimmedHistory []providers.Message) []providers.Message {
				fullHistory := append(append([]providers.Message(nil), trimmedHistory...), protectedTurnTail...)
				rebuildPromptReq := promptBuildRequestForTurn(ts, fullHistory, exec.summary, "", nil, p.Cfg)
				rebuildPromptReq.ActiveSkills = append([]string(nil), contextualSkills...)
				rebuilt := ts.agent.ContextBuilder.BuildMessagesFromPrompt(rebuildPromptReq)
				return resolveMediaRefs(
					rebuilt,
					p.MediaStore,
					maxMediaSize,
					len(rebuilt)-len(protectedTurnTail),
				)
			}
			originalHistoryCount := len(exec.history)
			var fit bool
			var trimmedStableHistory []providers.Message
			trimmedStableHistory, exec.callMessages, fit = trimHistoryToFitContextWindow(
				stableHistory,
				func(trimmedHistory []providers.Message) []providers.Message {
					rebuilt := buildMessages(trimmedHistory)
					if exec.gracefulTerminal {
						return append(append([]providers.Message(nil), rebuilt...), ts.interruptHintMessage())
					}
					return rebuilt
				},
				ts.agent.ContextWindow,
				exec.providerToolDefs,
				ts.agent.MaxTokens,
			)
			exec.history = append(trimmedStableHistory, protectedTurnTail...)
			exec.messages = buildMessages(trimmedStableHistory)
			exec.currentTurnStart = len(exec.messages) - len(protectedTurnTail)
			if exec.gracefulTerminal {
				msgs := append([]providers.Message(nil), exec.messages...)
				exec.callMessages = append(msgs, ts.interruptHintMessage())
			}
			if dropped := originalHistoryCount - len(exec.history); dropped > 0 {
				logger.WarnCF("agent", "Trimmed rebuilt history after context retry compaction", map[string]any{
					"session_key":     ts.sessionKey,
					"retry":           retry,
					"dropped_msgs":    dropped,
					"remaining_msgs":  len(exec.history),
					"context_window":  ts.agent.ContextWindow,
					"max_tokens":      ts.agent.MaxTokens,
					"still_overlimit": !fit,
				})
			} else if !fit {
				logger.WarnCF("agent", "Context still exceeds budget after retry compaction rebuild", map[string]any{
					"session_key":         ts.sessionKey,
					"retry":               retry,
					"history_msgs":        len(exec.history),
					"protected_turn_msgs": len(protectedTurnTail),
					"context_window":      ts.agent.ContextWindow,
					"max_tokens":          ts.agent.MaxTokens,
				})
			}
			if !fit {
				err = fmt.Errorf(
					"context window still exceeded after retry compaction; refusing to drop active turn messages: %w",
					err,
				)
				break
			}
			continue
		}
		break
	}

	if err != nil {
		al.emitEvent(
			runtimeevents.KindAgentError,
			ts.eventMeta("runTurn", "turn.error"),
			ErrorPayload{
				Stage:   "llm",
				Message: err.Error(),
			},
		)
		logger.ErrorCF("agent", "LLM call failed",
			map[string]any{
				"agent_id":  ts.agent.ID,
				"iteration": iteration,
				"model":     exec.llmModel,
				"error":     err.Error(),
			})
		p.emitProviderFailoverExhausted(ts, exec, err)
		return ControlBreak, fmt.Errorf("LLM call failed after retries: %w", err)
	}

	// AfterLLM hook
	if p.Hooks != nil {
		llmResp, decision := p.Hooks.AfterLLM(turnCtx, &LLMHookResponse{
			Meta:     ts.eventMeta("runTurn", "turn.llm.response"),
			Context:  cloneTurnContext(ts.turnCtx),
			Model:    exec.llmModel,
			Response: exec.response,
		})
		switch decision.normalizedAction() {
		case HookActionContinue, HookActionModify:
			if llmResp != nil && llmResp.Response != nil {
				exec.response = llmResp.Response
			}
		case HookActionAbortTurn:
			cancelConfiguredStreamingLLM(turnCtx, exec)
			exec.abortedByHook = true
			return ControlBreak, nil
		case HookActionHardAbort:
			cancelConfiguredStreamingLLM(turnCtx, exec)
			_ = ts.requestHardAbort()
			exec.abortedByHardAbort = true
			return ControlBreak, nil
		}
	}

	// Save finishReason to turnState for SubTurn truncation detection
	if innerTS := turnStateFromContext(ctx); innerTS != nil {
		innerTS.SetLastFinishReason(exec.response.FinishReason)
		if exec.response.Usage != nil {
			innerTS.SetLastUsage(exec.response.Usage)
		}
	}

	if exec.suppressReasoning {
		exec.response.Reasoning = ""
		exec.response.ReasoningContent = ""
		exec.response.ReasoningDetails = nil
	}
	reasoningContent := responseReasoningContent(exec.response)
	shouldPublishPocketClawToolCallInterim := ts.channel == config.ChannelPocketClaw && len(exec.response.ToolCalls) > 0
	if shouldPublishPocketClawToolCallInterim {
		// Pico tool-call turns publish their reasoning/content/tool summary as a
		// structured sequence after the tool-call payload is normalized below.
	} else if ts.channel == config.ChannelPocketClaw {
		if exec.streamingPublisher != nil && exec.streamingPublisher.ReasoningPublished() {
			if err := exec.streamingPublisher.FinalizeReasoning(turnCtx, reasoningContent); err != nil {
				logger.WarnCF("agent", "Failed to finalize streamed pico reasoning", map[string]any{
					"channel": ts.channel,
					"chat_id": ts.chatID,
					"error":   err.Error(),
				})
			}
		} else {
			// Publish pico thoughts before the turn context is canceled at return time.
			// The async variant can race with turn teardown and intermittently drop the
			// thought message in CI even though the LLM produced reasoning content.
			al.publishPicoReasoning(turnCtx, reasoningContent, ts.chatID, ts.sessionKey, exec.llmModelName)
		}
	} else {
		go al.handleReasoning(
			turnCtx,
			reasoningContent,
			ts.channel,
			al.targetReasoningChannelID(ts.channel),
		)
	}
	al.emitEvent(
		runtimeevents.KindAgentLLMResponse,
		ts.eventMeta("runTurn", "turn.llm.response"),
		LLMResponsePayload{
			ContentLen:   len(exec.response.Content),
			ToolCalls:    len(exec.response.ToolCalls),
			HasReasoning: exec.response.Reasoning != "" || exec.response.ReasoningContent != "",
		},
	)

	llmResponseFields := map[string]any{
		"agent_id":          ts.agent.ID,
		"iteration":         iteration,
		"content_chars":     len(exec.response.Content),
		"tool_calls":        len(exec.response.ToolCalls),
		"reasoning_present": exec.response.Reasoning != "" || exec.response.ReasoningContent != "",
		"reasoning_chars":   len(exec.response.Reasoning) + len(exec.response.ReasoningContent),
		"target_channel":    al.targetReasoningChannelID(ts.channel),
		"channel":           ts.channel,
	}
	if exec.response.Usage != nil {
		llmResponseFields["prompt_tokens"] = exec.response.Usage.PromptTokens
		llmResponseFields["completion_tokens"] = exec.response.Usage.CompletionTokens
		llmResponseFields["total_tokens"] = exec.response.Usage.TotalTokens
	}
	logger.DebugCF("agent", "LLM response", llmResponseFields)

	// No-tool-call path: steering check and direct response
	if len(exec.response.ToolCalls) == 0 || exec.gracefulTerminal {
		responseContent := exec.response.Content
		if responseContent == "" && exec.response.ReasoningContent != "" && ts.channel != config.ChannelPocketClaw {
			responseContent = exec.response.ReasoningContent
		}
		if steerMsgs := al.dequeueSteeringMessagesForScope(ts.sessionKey); len(steerMsgs) > 0 {
			cancelConfiguredStreamingLLM(turnCtx, exec)
			logger.InfoCF("agent", "Steering arrived after direct LLM response; continuing turn",
				map[string]any{
					"agent_id":       ts.agent.ID,
					"iteration":      iteration,
					"steering_count": len(steerMsgs),
				})
			exec.pendingMessages = append(exec.pendingMessages, steerMsgs...)
			return ControlContinue, nil
		}

		exec.finalContent = responseContent
		logger.InfoCF("agent", "LLM response without tool calls (direct answer)",
			map[string]any{
				"agent_id":      ts.agent.ID,
				"iteration":     iteration,
				"content_chars": len(exec.finalContent),
			})
		return ControlBreak, nil
	}
	cancelConfiguredStreamingLLM(turnCtx, exec)

	// Tool-call path: normalize and prepare for tool execution
	exec.normalizedToolCalls = make([]providers.ToolCall, 0, len(exec.response.ToolCalls))
	for _, tc := range exec.response.ToolCalls {
		exec.normalizedToolCalls = append(exec.normalizedToolCalls, providers.NormalizeToolCall(tc))
	}

	toolNames := make([]string, 0, len(exec.normalizedToolCalls))
	for _, tc := range exec.normalizedToolCalls {
		toolNames = append(toolNames, tc.Name)
		traceTurnLifecycle("tool_requested", ts, map[string]any{
			"tool": tc.Name,
		})
	}
	logger.InfoCF("agent", "LLM requested tool calls",
		map[string]any{
			"agent_id":  ts.agent.ID,
			"tools":     toolNames,
			"count":     len(exec.normalizedToolCalls),
			"iteration": iteration,
		})

	exec.allResponsesHandled = len(exec.normalizedToolCalls) > 0
	assistantMsg := providers.Message{
		Role:             "assistant",
		Content:          exec.response.Content,
		ModelName:        exec.llmModelName,
		ReasoningContent: reasoningContent,
	}
	for _, tc := range exec.normalizedToolCalls {
		argumentsJSON, _ := json.Marshal(tc.Arguments)
		toolFeedbackExplanation := toolFeedbackExplanationForToolCall(
			exec.response,
			tc,
			exec.messages,
		)
		extraContent := tc.ExtraContent
		if strings.TrimSpace(toolFeedbackExplanation) != "" {
			if extraContent == nil {
				extraContent = &providers.ExtraContent{}
			}
			extraContent.ToolFeedbackExplanation = toolFeedbackExplanation
		}
		thoughtSignature := ""
		if tc.Function != nil {
			thoughtSignature = tc.Function.ThoughtSignature
		}
		assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, providers.ToolCall{
			ID:   tc.ID,
			Type: "function",
			Name: tc.Name,
			Function: &providers.FunctionCall{
				Name:             tc.Name,
				Arguments:        string(argumentsJSON),
				ThoughtSignature: thoughtSignature,
			},
			ExtraContent:     extraContent,
			ThoughtSignature: thoughtSignature,
		})
	}
	exec.messages = append(exec.messages, assistantMsg)
	if !ts.opts.NoHistory {
		ts.agent.Sessions.AddFullMessage(ts.sessionKey, assistantMsg)
		ts.recordPersistedMessage(assistantMsg)
		ts.ingestMessage(turnCtx, al, assistantMsg)
	}
	if shouldPublishPocketClawToolCallInterim {
		al.publishPicoToolCallInterim(
			turnCtx,
			ts,
			exec.llmModelName,
			reasoningContent,
			exec.response.Content,
			assistantMsg.ToolCalls,
		)
	}

	return ControlToolLoop, nil
}

func (p *Pipeline) applyBeforeLLMModelRewrite(ts *turnState, exec *turnExecution) {
	if p == nil || ts == nil || ts.agent == nil || exec == nil {
		return
	}
	rawModel := strings.TrimSpace(exec.llmModel)
	if rawModel == "" {
		return
	}

	defaultProvider := "openai"
	if p.Cfg != nil {
		if provider := strings.TrimSpace(p.Cfg.Agents.Defaults.Provider); provider != "" {
			defaultProvider = provider
		}
	}
	defaultProvider = effectiveDefaultProvider(defaultProvider)
	candidates := resolveModelCandidates(p.Cfg, defaultProvider, rawModel, nil)
	exec.activeCandidates = candidates
	exec.activeModel = resolvedCandidateModel(candidates, rawModel)
	exec.llmModel = exec.activeModel
	exec.activeModelConfig = resolveActiveModelConfig(p.Cfg, ts.agent.Workspace, candidates, rawModel, defaultProvider)
}

// providerForFallbackCandidate resolves the provider a candidate must be called
// through.
//
// The lookup is by the candidate's stable identity, not by provider/model: two
// model_list entries can name the same protocol and model id, and keying on
// that pair let one of them answer for the other. That is how configuring a
// fallback changed the request sent to the primary.
func providerForFallbackCandidate(
	agent *AgentInstance,
	activeProvider providers.LLMProvider,
	activeCandidates []providers.FallbackCandidate,
	candidate providers.FallbackCandidate,
) (providers.LLMProvider, error) {
	if agent != nil {
		if cp, ok := agent.CandidateProviders[candidate.StableKey()]; ok && cp != nil {
			return cp, nil
		}
	}
	if activeProvider == nil {
		return nil, fmt.Errorf("fallback model %q has no active provider", candidate.Model)
	}
	return activeProvider, nil
}

func transientLLMRetryReason(err error) (string, bool) {
	if err == nil {
		return "", false
	}

	if failErr := providers.ClassifyError(err, "", ""); failErr != nil {
		switch failErr.Reason {
		case providers.FailoverTimeout:
			if failErr.Status >= 500 {
				return "server_error", true
			}
			return "timeout", true
		case providers.FailoverNetwork:
			return "network", true
		case providers.FailoverOverloaded:
			return "overloaded", true
		case providers.FailoverRateLimit:
			return "rate_limit", true
		case providers.FailoverHardQuota:
			// Reported for the log, but never retriable on this candidate: the
			// quota will not refill within the turn.
			return "hard_quota", false
		}
	}

	errMsg := strings.ToLower(err.Error())
	if errors.Is(err, context.DeadlineExceeded) ||
		strings.Contains(errMsg, "deadline exceeded") ||
		strings.Contains(errMsg, "client.timeout") ||
		strings.Contains(errMsg, "timed out") ||
		strings.Contains(errMsg, "timeout exceeded") {
		return "timeout", true
	}

	if strings.Contains(errMsg, "connection reset") ||
		strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "broken pipe") ||
		strings.Contains(errMsg, "no such host") ||
		strings.Contains(errMsg, "network is unreachable") ||
		strings.Contains(errMsg, "read tcp") ||
		strings.Contains(errMsg, "write tcp") ||
		strings.Contains(errMsg, "eof") {
		return "network", true
	}

	return "", false
}

// maxHonouredRetryAfter caps how long a provider's Retry-After may stall a turn
// when there is no alternative candidate. Beyond this the wait stops being
// recovery and becomes an unexplained hang.
const maxHonouredRetryAfter = 30 * time.Second

// classifyAndRecordProviderFailure classifies a provider error and records it
// against the shared cooldown state when no fallback chain did so.
//
// It returns the classification, or nil when the error is not attributable to
// the provider at all — an unclassifiable error must not put a healthy
// candidate into cooldown.
func (p *Pipeline) classifyAndRecordProviderFailure(
	ts *turnState,
	exec *turnExecution,
	err error,
	handledByChain bool,
) *providers.FailoverError {
	if err == nil || errors.Is(err, context.Canceled) {
		return nil
	}

	provider, model := providerAndModelForFailure(exec)
	failErr := providers.ClassifyError(err, provider, model)
	if failErr == nil {
		return nil
	}

	// The chain records its own failures per candidate; recording again here
	// would double-count and lengthen the backoff curve incorrectly.
	if handledByChain || p.Fallback == nil {
		return failErr
	}
	tracker := p.Fallback.Cooldown()
	if tracker == nil {
		return failErr
	}

	key := singleCandidateCooldownKey(exec, provider, model)
	if key == "" {
		return failErr
	}
	tracker.MarkFailureWithRetryAfter(key, failErr.Reason, failErr.RetryAfter)
	logger.DebugCF("agent", "Recorded provider failure for cooldown", map[string]any{
		"agent_id":    ts.agent.ID,
		"error_class": string(failErr.Reason),
		"candidate":   key,
	})
	return failErr
}

func providerAndModelForFailure(exec *turnExecution) (string, string) {
	if len(exec.activeCandidates) == 1 {
		return exec.activeCandidates[0].Provider, exec.activeCandidates[0].Model
	}
	return "", exec.llmModel
}

func singleCandidateCooldownKey(exec *turnExecution, provider, model string) string {
	if len(exec.activeCandidates) == 1 {
		return exec.activeCandidates[0].StableKey()
	}
	if provider == "" && model == "" {
		return ""
	}
	return providers.ModelKey(provider, model)
}

// responseIsUserVisiblyEmpty reports whether a response carries nothing for the
// user and nothing for the agent to act on.
//
// A response containing tool calls is emphatically not empty: it is the normal
// shape of every tool-using turn, and treating it as a failure would make the
// agent retry its own successful requests. Reasoning-only responses are equally
// not failures.
//
// This predicate exists for logging and for the end-of-turn placeholder. It is
// deliberately not wired to retry or failover.
func responseIsUserVisiblyEmpty(response *providers.LLMResponse) bool {
	if response == nil {
		return true
	}
	if len(response.ToolCalls) > 0 {
		return false
	}
	return strings.TrimSpace(response.Content) == ""
}

// providerAttemptFor builds the safe attempt payload for the candidate this
// call will actually use.
func (p *Pipeline) providerAttemptFor(
	ts *turnState,
	exec *turnExecution,
	attempt int,
) ProviderAttemptPayload {
	var candidate providers.FallbackCandidate
	if len(exec.activeCandidates) > 0 {
		candidate = exec.activeCandidates[0]
	} else {
		candidate = providers.FallbackCandidate{Model: exec.llmModel}
	}
	upstream := exec.llmModel
	if strings.TrimSpace(upstream) == "" {
		upstream = candidate.Model
	}
	return providerAttemptPayload(ts, candidate, upstream, attempt, 0)
}

// emitProviderFailoverExhausted records that every configured candidate failed.
// It is the one provider event a user is likely to see the consequence of, so
// the classified reason is carried rather than a bare failure.
func (p *Pipeline) emitProviderFailoverExhausted(
	ts *turnState,
	exec *turnExecution,
	err error,
) {
	payload := p.providerAttemptFor(ts, exec, 0)
	if failErr := providers.ClassifyError(err, payload.Provider, payload.UpstreamModel); failErr != nil {
		payload.ErrorClass = string(failErr.Reason)
		payload.HTTPStatus = failErr.Status
	}
	payload.FallbackIndex = max(0, len(exec.activeCandidates)-1)
	p.al.emitProviderEvent(
		runtimeevents.KindProviderFailoverExhaust,
		ts.eventMeta("runTurn", "provider.failover"),
		payload,
	)
}
