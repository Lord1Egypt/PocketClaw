import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'package:pocketclaw/src/core/picoclaw_channel.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';

/// The preset choices offered before Custom. Core accepts any value in its own
/// range; these are the ones worth one tap.
const List<int> contextMemoryPresets = <int>[10, 15, 20, 25];

/// Reads and writes the Telegram context-memory limit.
///
/// It changes one number. No message is deleted from Telegram, no stored
/// transcript or session history is touched, and the rolling summary is left
/// alone — only how many recent messages future turns project into the model.
///
/// Core owns `config.json` and validates the range, so this card asks Core for
/// the effective value and hands a new one back rather than editing the file.
class ContextMemoryCard extends StatefulWidget {
  const ContextMemoryCard({
    super.key,
    this.focusNode,
    this.saveFocusNode,
    this.prevFocusNode,
    this.nextFocusNode,
    this.load,
    this.save,
  });

  final FocusNode? focusNode;
  final FocusNode? saveFocusNode;
  final FocusNode? prevFocusNode;
  final FocusNode? nextFocusNode;

  /// Injected for tests so the card can be exercised without a platform channel.
  final Future<TelegramContextMemory> Function()? load;
  final Future<TelegramContextMemory> Function(int recentMessages)? save;

  @override
  State<ContextMemoryCard> createState() => _ContextMemoryCardState();
}

class _ContextMemoryCardState extends State<ContextMemoryCard> {
  TelegramContextMemory _setting = TelegramContextMemory.unknown;
  final TextEditingController _customController = TextEditingController();
  bool _customSelected = false;
  bool _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _loadSetting();
  }

  @override
  void dispose() {
    _customController.dispose();
    super.dispose();
  }

  Future<void> _loadSetting() async {
    final loader = widget.load ?? PicoClawChannel.getTelegramContextMemory;
    TelegramContextMemory loaded;
    try {
      loaded = await loader();
    } catch (_) {
      // Core may not be running yet. The shipped default is shown rather than
      // an error, because nothing is wrong until the user tries to save.
      loaded = TelegramContextMemory.unknown;
    }
    if (!mounted) return;
    setState(() {
      _setting = loaded;
      _customSelected = !contextMemoryPresets.contains(loaded.recentMessages);
      _customController.text = loaded.recentMessages.toString();
      _loading = false;
    });
  }

  Future<void> _apply(int value) async {
    final l10n = AppLocalizations.of(context)!;
    if (!_setting.accepts(value)) {
      setState(() {
        _error = l10n.contextMemoryRangeError(_setting.min, _setting.max);
      });
      return;
    }

    final previous = _setting;
    setState(() {
      _setting = _setting.copyWith(recentMessages: value);
      _customSelected = !contextMemoryPresets.contains(value);
      _error = null;
    });

    final saver = widget.save ?? PicoClawChannel.setTelegramContextMemory;
    try {
      final stored = await saver(value);
      if (!mounted) return;
      setState(() {
        _setting = stored;
        _customSelected = !contextMemoryPresets.contains(stored.recentMessages);
        _customController.text = stored.recentMessages.toString();
      });
    } catch (_) {
      // Core rejected or could not be reached. The previous value is restored
      // so the card never shows a setting Core is not actually using.
      if (!mounted) return;
      setState(() {
        _setting = previous;
        _customSelected = !contextMemoryPresets.contains(
          previous.recentMessages,
        );
        _customController.text = previous.recentMessages.toString();
        _error = l10n.contextMemorySaveFailed;
      });
    }
  }

  void _selectCustom() {
    setState(() {
      _customSelected = true;
      _error = null;
    });
  }

  /// The value the field currently holds, or null when it is not a usable
  /// number. Save and Enter both go through this, so one action is one save.
  int? get _pendingCustomValue => int.tryParse(_customController.text.trim());

  /// Save is offered only when pressing it would do something: the field holds
  /// a value Core will accept, and that value is not the one already stored.
  bool get _canSaveCustom {
    final pending = _pendingCustomValue;
    return pending != null &&
        _setting.accepts(pending) &&
        pending != _setting.recentMessages;
  }

  Future<void> _submitCustom(String raw) async {
    final l10n = AppLocalizations.of(context)!;
    final parsed = int.tryParse(raw.trim());
    if (parsed == null) {
      setState(() {
        _error = l10n.contextMemoryRangeError(_setting.min, _setting.max);
      });
      return;
    }
    await _apply(parsed);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;

    return Card(
      key: const Key('context-memory-card'),
      margin: EdgeInsets.zero,
      child: Padding(
        padding: const EdgeInsetsDirectional.fromSTEB(16, 12, 16, 16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.psychology_outlined, color: scheme.primary),
                const SizedBox(width: 12),
                Expanded(
                  child: Text(
                    l10n.contextMemoryTitle,
                    style: theme.textTheme.titleMedium,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              l10n.contextMemoryDescription,
              style: theme.textTheme.bodySmall,
            ),
            const SizedBox(height: 12),
            // The chips are always laid out, disabled until Core has answered.
            // An indeterminate progress indicator would animate forever while
            // the value loads, which never settles and is more motion than a
            // number that arrives immediately deserves.
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                for (final preset in contextMemoryPresets)
                  ChoiceChip(
                    key: Key('context-memory-preset-$preset'),
                    focusNode: preset == contextMemoryPresets.first
                        ? widget.focusNode
                        : null,
                    selected:
                        !_loading &&
                        !_customSelected &&
                        _setting.recentMessages == preset,
                    label: Text(
                      preset == _setting.defaultValue
                          ? '$preset · ${l10n.contextMemoryRecommended}'
                          : '$preset',
                    ),
                    onSelected: _loading ? null : (_) => _apply(preset),
                  ),
                ChoiceChip(
                  key: const Key('context-memory-preset-custom'),
                  selected: !_loading && _customSelected,
                  label: Text(l10n.contextMemoryCustom),
                  onSelected: _loading ? null : (_) => _selectCustom(),
                ),
              ],
            ),
            if (_customSelected && !_loading) ...[
              const SizedBox(height: 12),
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  SizedBox(
                    width: 200,
                    child: TextField(
                      key: const Key('context-memory-custom-field'),
                      controller: _customController,
                      keyboardType: TextInputType.number,
                      inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                      decoration: InputDecoration(
                        labelText: l10n.contextMemoryCustomLabel,
                        border: const OutlineInputBorder(),
                        isDense: true,
                        helperText: '${_setting.min}–${_setting.max}',
                      ),
                      // Typing only re-evaluates whether Save is offered; it
                      // never saves. Only onSubmitted commits — wiring
                      // onEditingComplete as well makes one confirmation save
                      // twice.
                      onChanged: (_) => setState(() => _error = null),
                      onSubmitted: _submitCustom,
                    ),
                  ),
                  const SizedBox(width: 12),
                  Padding(
                    padding: const EdgeInsetsDirectional.only(top: 2),
                    child: FilledButton(
                      key: const Key('context-memory-save'),
                      focusNode: widget.saveFocusNode,
                      onPressed: _canSaveCustom
                          ? () => _submitCustom(_customController.text)
                          : null,
                      child: Text(l10n.settingsSave),
                    ),
                  ),
                ],
              ),
            ],
            if (_error != null) ...[
              const SizedBox(height: 8),
              Text(
                _error!,
                key: const Key('context-memory-error'),
                style: theme.textTheme.bodySmall?.copyWith(color: scheme.error),
              ),
            ],
            const SizedBox(height: 8),
            Text(
              l10n.contextMemoryHelp,
              style: theme.textTheme.bodySmall?.copyWith(
                color: scheme.onSurface.withAlpha(160),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
