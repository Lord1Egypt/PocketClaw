import 'dart:async';
import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:uuid/uuid.dart';
import 'package:web_socket_channel/web_socket_channel.dart';
import 'package:web_socket_channel/io.dart';
import 'package:picoclaw_flutter_ui/src/core/picoclaw_channel.dart';

/// 聊天页面 - 通过 WebSocket 与 PicoClaw Gateway 的 Pico Protocol 通信
class ChatPage extends StatefulWidget {
  const ChatPage({super.key});

  @override
  State<ChatPage> createState() => _ChatPageState();
}

class _ChatPageState extends State<ChatPage> {
  static const _gatewayHost = '127.0.0.1';
  static const _gatewayPort = 18790;
  static const _keySessionId = 'session_id';

  final _messageController = TextEditingController();
  final _scrollController = ScrollController();
  final _messages = <_ChatMessage>[];
  final _uuid = const Uuid();

  WebSocketChannel? _channel;
  StreamSubscription? _subscription;
  bool _isConnected = false;
  bool _isSending = false;
  late String _sessionId;
  String _picoToken = '';

  @override
  void initState() {
    super.initState();
    _initChat();
  }

  Future<void> _initChat() async {
    _sessionId = await _getOrCreateSessionId();
    try {
      _picoToken = await PicoClawChannel.getPicoToken();
    } catch (_) {
      _picoToken = 'picoclaw-android-local';
    }
    _addMessage(_ChatMessage('正在连接 AI 助手...', _Role.assistant));
    _connectToGateway();
  }

  Future<String> _getOrCreateSessionId() async {
    final prefs = await SharedPreferences.getInstance();
    var id = prefs.getString(_keySessionId);
    if (id == null) {
      id = _uuid.v4();
      await prefs.setString(_keySessionId, id);
    }
    return id;
  }

  void _connectToGateway() {
    try {
      final uri = Uri.parse(
        'ws://$_gatewayHost:$_gatewayPort/pico/ws?session_id=$_sessionId',
      );
      _channel = IOWebSocketChannel.connect(
        uri,
        headers: {'Authorization': 'Bearer $_picoToken'},
      );

      _subscription = _channel!.stream.listen(
        (data) => _handleGatewayMessage(data.toString()),
        onDone: () {
          if (mounted) {
            setState(() {
              _isConnected = false;
              _isSending = false;
            });
            _addMessage(_ChatMessage('⚠️ 连接已断开', _Role.assistant));
          }
        },
        onError: (error) {
          if (mounted) {
            setState(() {
              _isConnected = false;
              _isSending = false;
            });
            _addMessage(
              _ChatMessage(
                '❌ 连接失败: $error\n请确保 PicoClaw 服务正在运行。',
                _Role.assistant,
              ),
            );
          }
        },
      );

      // WebSocketChannel.connect 成功后不会触发 onOpen 回调，
      // 通过监听 ready future 来确认连接成功
      _channel!.ready
          .then((_) {
            if (mounted) {
              setState(() {
                _isConnected = true;
                _messages.clear();
              });
              _addMessage(
                _ChatMessage('你好！我是 AI 助手，有什么可以帮你的？', _Role.assistant),
              );
            }
          })
          .catchError((e) {
            if (mounted) {
              setState(() {
                _isConnected = false;
              });
              _addMessage(_ChatMessage('❌ 连接失败: $e', _Role.assistant));
            }
          });
    } catch (e) {
      _addMessage(_ChatMessage('❌ 连接错误: $e', _Role.assistant));
    }
  }

  void _handleGatewayMessage(String text) {
    try {
      final msg = jsonDecode(text) as Map<String, dynamic>;
      final type = msg['type'] as String? ?? '';
      final payload = msg['payload'] as Map<String, dynamic>?;

      switch (type) {
        case 'message.create':
          final content = payload?['content'] as String? ?? '';
          if (content.isNotEmpty) {
            setState(() {
              _removeThinkingMessage();
              _isSending = false;
            });
            _addMessage(_ChatMessage(content, _Role.assistant));
          }
          break;
        case 'message.update':
          final content = payload?['content'] as String? ?? '';
          if (content.isNotEmpty) {
            setState(() {
              _removeThinkingMessage();
            });
            _addMessage(_ChatMessage(content, _Role.assistant));
          }
          break;
        case 'typing.start':
          if (!_messages.any((m) => m.isThinking)) {
            _addMessage(
              _ChatMessage('正在思考...', _Role.assistant, isThinking: true),
            );
          }
          break;
        case 'typing.stop':
          // typing stop 由 message.create 隐式处理
          break;
        case 'error':
          final errorMsg = payload?['message'] as String? ?? '未知错误';
          setState(() {
            _removeThinkingMessage();
            _isSending = false;
          });
          _addMessage(_ChatMessage('❌ 错误: $errorMsg', _Role.assistant));
          break;
        case 'pong':
          break; // 忽略心跳
      }
    } catch (e) {
      _addMessage(_ChatMessage('❌ 解析消息失败: $e', _Role.assistant));
    }
  }

  void _removeThinkingMessage() {
    _messages.removeWhere((m) => m.isThinking);
  }

  void _addMessage(_ChatMessage message) {
    if (!mounted) return;
    setState(() {
      _messages.add(message);
    });
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scrollController.hasClients) {
        _scrollController.animateTo(
          _scrollController.position.maxScrollExtent,
          duration: const Duration(milliseconds: 200),
          curve: Curves.easeOut,
        );
      }
    });
  }

  void _sendMessage() {
    final text = _messageController.text.trim();
    if (text.isEmpty) return;

    if (!_isConnected) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('未连接到 AI 服务，正在重连...')));
      _connectToGateway();
      return;
    }

    _messageController.clear();
    _addMessage(_ChatMessage(text, _Role.user));
    _addMessage(_ChatMessage('正在思考...', _Role.assistant, isThinking: true));
    setState(() => _isSending = true);

    // 通过 WebSocket 发送 Pico Protocol 消息
    final picoMsg = jsonEncode({
      'type': 'message.send',
      'id': _uuid.v4(),
      'session_id': _sessionId,
      'payload': {'content': text},
    });

    try {
      _channel?.sink.add(picoMsg);
    } catch (e) {
      setState(() {
        _removeThinkingMessage();
        _isSending = false;
      });
      _addMessage(_ChatMessage('❌ 发送失败，请重试', _Role.assistant));
    }
  }

  void _clearChat() async {
    final prefs = await SharedPreferences.getInstance();
    _sessionId = _uuid.v4();
    await prefs.setString(_keySessionId, _sessionId);

    setState(() {
      _messages.clear();
      _isSending = false;
    });

    _subscription?.cancel();
    _channel?.sink.close();
    _addMessage(_ChatMessage('正在开始新对话...', _Role.assistant));
    _connectToGateway();
  }

  @override
  void dispose() {
    _subscription?.cancel();
    _channel?.sink.close();
    _messageController.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return Scaffold(
      appBar: AppBar(
        title: Text(
          'AI Chat',
          style: GoogleFonts.inter(fontWeight: FontWeight.w700),
        ),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => Navigator.of(context).pop(),
        ),
        actions: [
          // 连接状态指示
          Container(
            margin: const EdgeInsets.only(right: 8),
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
            decoration: BoxDecoration(
              color: (_isConnected ? Colors.green : Colors.red).withAlpha(30),
              borderRadius: BorderRadius.circular(12),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Container(
                  width: 6,
                  height: 6,
                  decoration: BoxDecoration(
                    color: _isConnected ? Colors.green : Colors.red,
                    shape: BoxShape.circle,
                  ),
                ),
                const SizedBox(width: 4),
                Text(
                  _isConnected ? '已连接' : '未连接',
                  style: TextStyle(
                    fontSize: 11,
                    color: _isConnected ? Colors.green : Colors.red,
                  ),
                ),
              ],
            ),
          ),
          IconButton(
            icon: const Icon(Icons.delete_outline),
            tooltip: '清空对话',
            onPressed: _clearChat,
          ),
        ],
      ),
      body: Column(
        children: [
          // 消息列表
          Expanded(
            child: ListView.builder(
              controller: _scrollController,
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
              itemCount: _messages.length,
              itemBuilder: (context, index) {
                final message = _messages[index];
                return _MessageBubble(message: message);
              },
            ),
          ),
          // 输入区域
          Container(
            decoration: BoxDecoration(
              color: colorScheme.surface,
              border: Border(
                top: BorderSide(
                  color: colorScheme.outlineVariant.withAlpha(50),
                ),
              ),
            ),
            padding: const EdgeInsets.fromLTRB(16, 8, 8, 8),
            child: SafeArea(
              child: Row(
                children: [
                  Expanded(
                    child: TextField(
                      controller: _messageController,
                      decoration: InputDecoration(
                        hintText: '输入消息...',
                        border: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(24),
                          borderSide: BorderSide.none,
                        ),
                        filled: true,
                        fillColor: colorScheme.surfaceContainerHigh,
                        contentPadding: const EdgeInsets.symmetric(
                          horizontal: 20,
                          vertical: 10,
                        ),
                      ),
                      textInputAction: TextInputAction.send,
                      onSubmitted: (_) => _sendMessage(),
                      maxLines: null,
                    ),
                  ),
                  const SizedBox(width: 8),
                  Material(
                    color: _isSending
                        ? colorScheme.outline
                        : colorScheme.secondary,
                    borderRadius: BorderRadius.circular(24),
                    child: InkWell(
                      onTap: _isSending ? null : _sendMessage,
                      borderRadius: BorderRadius.circular(24),
                      child: Container(
                        padding: const EdgeInsets.all(12),
                        child: Icon(
                          Icons.send_rounded,
                          color: colorScheme.onSecondary,
                          size: 20,
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

// --- 数据模型 ---

enum _Role { user, assistant }

class _ChatMessage {
  final String content;
  final _Role role;
  final bool isThinking;

  _ChatMessage(this.content, this.role, {this.isThinking = false});
}

// --- 消息气泡组件 ---

class _MessageBubble extends StatelessWidget {
  final _ChatMessage message;

  const _MessageBubble({required this.message});

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final isUser = message.role == _Role.user;

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        mainAxisAlignment: isUser
            ? MainAxisAlignment.end
            : MainAxisAlignment.start,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (!isUser) ...[
            CircleAvatar(
              radius: 16,
              backgroundColor: colorScheme.secondary.withAlpha(30),
              child: Icon(
                Icons.smart_toy_outlined,
                size: 18,
                color: colorScheme.secondary,
              ),
            ),
            const SizedBox(width: 8),
          ],
          Flexible(
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
              decoration: BoxDecoration(
                color: isUser
                    ? colorScheme.secondary.withAlpha(40)
                    : colorScheme.surfaceContainerHigh,
                borderRadius: BorderRadius.only(
                  topLeft: const Radius.circular(16),
                  topRight: const Radius.circular(16),
                  bottomLeft: Radius.circular(isUser ? 16 : 4),
                  bottomRight: Radius.circular(isUser ? 4 : 16),
                ),
              ),
              child: message.isThinking
                  ? Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        SizedBox(
                          width: 14,
                          height: 14,
                          child: CircularProgressIndicator(
                            strokeWidth: 2,
                            color: colorScheme.secondary,
                          ),
                        ),
                        const SizedBox(width: 8),
                        Text(
                          message.content,
                          style: TextStyle(
                            color: colorScheme.onSurface.withAlpha(150),
                            fontStyle: FontStyle.italic,
                          ),
                        ),
                      ],
                    )
                  : SelectableText(
                      message.content,
                      style: TextStyle(
                        color: colorScheme.onSurface,
                        height: 1.4,
                      ),
                    ),
            ),
          ),
          if (isUser) ...[
            const SizedBox(width: 8),
            CircleAvatar(
              radius: 16,
              backgroundColor: colorScheme.primary.withAlpha(30),
              child: Icon(
                Icons.person_outline,
                size: 18,
                color: colorScheme.primary,
              ),
            ),
          ],
        ],
      ),
    );
  }
}
