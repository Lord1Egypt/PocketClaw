import 'package:flutter/material.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';

/// Focus highlight for keyboard and D-pad navigation: border, scale and glow.
/// On a touch screen focus rarely moves, so the effect seldom shows there.
class TVFocusable extends StatefulWidget {
  final Widget child;
  final VoidCallback? onTap;
  final BorderRadius? borderRadius;
  final double focusBorderWidth;
  final Color? focusBorderColor;
  final double focusScale;
  final Color? focusBackgroundColor;
  final bool showFocusGlow;
  final bool autofocus;
  final EdgeInsetsGeometry? focusPadding;

  const TVFocusable({
    super.key,
    required this.child,
    this.onTap,
    this.borderRadius,
    this.focusBorderWidth = 2.0,
    this.focusBorderColor,
    this.focusScale = 1.01,
    this.focusBackgroundColor,
    this.showFocusGlow = true,
    this.autofocus = false,
    this.focusPadding,
  });

  @override
  State<TVFocusable> createState() => _TVFocusableState();
}

class _TVFocusableState extends State<TVFocusable> {
  bool _isFocused = false;
  final FocusNode _focusNode = FocusNode();

  @override
  void dispose() {
    _focusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final focusColor = widget.focusBorderColor ?? colorScheme.secondary;
    final bgColor =
        widget.focusBackgroundColor ??
        focusColor.withAlpha((0.08 * 255).round());

    final effectiveBorderWidth = widget.focusBorderWidth;
    final effectiveFocusScale = widget.focusScale;
    final effectiveShowGlow = widget.showFocusGlow;

    return FocusableActionDetector(
      focusNode: _focusNode,
      autofocus: widget.autofocus,
      onFocusChange: (focused) {
        setState(() => _isFocused = focused);
      },
      actions: {
        ActivateIntent: CallbackAction<ActivateIntent>(
          onInvoke: (_) {
            widget.onTap?.call();
            return null;
          },
        ),
      },
      child: GestureDetector(
        onTap: () {
          // On a touch device a tap also takes focus.
          if (!_focusNode.hasFocus) {
            _focusNode.requestFocus();
          }
          widget.onTap?.call();
        },
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 150),
          curve: Curves.easeOut,
          margin: _isFocused ? widget.focusPadding : EdgeInsets.zero,
          decoration: BoxDecoration(
            borderRadius: widget.borderRadius ?? BorderRadius.circular(12),
            color: _isFocused ? bgColor : null,
            boxShadow: _isFocused && effectiveShowGlow
                ? [
                    BoxShadow(
                      color: focusColor.withAlpha(
                        ((0.15).clamp(0.0, 1.0) * 255).round(),
                      ),
                      blurRadius: 8,
                      spreadRadius: 0,
                    ),
                  ]
                : null,
            border: _isFocused
                ? Border.all(
                    color: focusColor.withAlpha((0.6 * 255).round()),
                    width: effectiveBorderWidth,
                  )
                : null,
          ),
          child: AnimatedScale(
            scale: _isFocused ? effectiveFocusScale : 1.0,
            duration: const Duration(milliseconds: 150),
            curve: Curves.easeOut,
            child: widget.child,
          ),
        ),
      ),
    );
  }
}

/// A text field that can be reached with a TV remote.
class TVTextField extends StatelessWidget {
  final TextEditingController controller;
  final String? labelText;
  final String? hintText;
  final bool enabled;
  final bool readOnly;
  final TextInputType? keyboardType;
  final int? maxLines;
  final VoidCallback? onTap;
  final ValueChanged<String>? onChanged;
  final bool autofocus;

  const TVTextField({
    super.key,
    required this.controller,
    this.labelText,
    this.hintText,
    this.enabled = true,
    this.readOnly = false,
    this.keyboardType,
    this.maxLines = 1,
    this.onTap,
    this.onChanged,
    this.autofocus = false,
  });

  @override
  Widget build(BuildContext context) {
    return TVFocusable(
      onTap:
          onTap ??
          (enabled && !readOnly ? () => _showEditDialog(context) : null),
      borderRadius: BorderRadius.circular(8),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            if (labelText != null)
              Text(
                labelText!,
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                  color: enabled
                      ? Theme.of(context).colorScheme.primary
                      : Theme.of(context).colorScheme.outline,
                ),
              ),
            const SizedBox(height: 4),
            Text(
              controller.text.isEmpty ? (hintText ?? '') : controller.text,
              style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                color: controller.text.isEmpty
                    ? Theme.of(context).colorScheme.outline
                    : Theme.of(context).colorScheme.onSurface,
              ),
              maxLines: maxLines,
              overflow: TextOverflow.ellipsis,
            ),
          ],
        ),
      ),
    );
  }

  void _showEditDialog(BuildContext context) {
    final textController = TextEditingController(text: controller.text);
    final l10n = AppLocalizations.of(context)!;
    showDialog(
      context: context,
      builder: (ctx) {
        final colorScheme = Theme.of(ctx).colorScheme;
        final btnStyle = TextStyle(color: colorScheme.secondary);
        return AlertDialog(
          title: Text(
            labelText ?? 'Edit',
            style: TextStyle(
              color: colorScheme.secondary,
              fontSize: 20,
              fontWeight: FontWeight.bold,
            ),
          ),
          content: TextField(
            controller: textController,
            keyboardType: keyboardType,
            maxLines: maxLines,
            autofocus: true,
            style: TextStyle(color: colorScheme.secondary),
            decoration: InputDecoration(hintText: hintText),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(ctx),
              child: Text(l10n.cancel, style: btnStyle),
            ),
            TextButton(
              onPressed: () {
                controller.text = textController.text;
                onChanged?.call(textController.text);
                Navigator.pop(ctx);
              },
              child: Text(l10n.save, style: btnStyle),
            ),
          ],
        );
      },
    );
  }
}

/// A switch that can be reached with a TV remote.
class TVSwitch extends StatelessWidget {
  final bool value;
  final ValueChanged<bool> onChanged;
  final String? title;
  final String? subtitle;
  final bool autofocus;

  const TVSwitch({
    super.key,
    required this.value,
    required this.onChanged,
    this.title,
    this.subtitle,
    this.autofocus = false,
  });

  @override
  Widget build(BuildContext context) {
    return TVFocusable(
      autofocus: autofocus,
      onTap: () => onChanged(!value),
      borderRadius: BorderRadius.circular(12),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
        child: Row(
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  if (title != null)
                    Text(title!, style: Theme.of(context).textTheme.bodyLarge),
                  if (subtitle != null)
                    Text(
                      subtitle!,
                      style: Theme.of(context).textTheme.bodySmall?.copyWith(
                        color: Theme.of(context).colorScheme.outline,
                      ),
                    ),
                ],
              ),
            ),
            Switch(value: value, onChanged: onChanged),
          ],
        ),
      ),
    );
  }
}

/// A button that can be reached with a TV remote.
class TVButton extends StatelessWidget {
  final VoidCallback onPressed;
  final Widget child;
  final ButtonStyle? style;
  final bool autofocus;

  const TVButton({
    super.key,
    required this.onPressed,
    required this.child,
    this.style,
    this.autofocus = false,
  });

  @override
  Widget build(BuildContext context) {
    return TVFocusable(
      autofocus: autofocus,
      onTap: onPressed,
      borderRadius: BorderRadius.circular(10),
      child: ElevatedButton(onPressed: onPressed, style: style, child: child),
    );
  }
}

/// An option-card selector that can be reached with a TV remote.
class TVCardSelector<T> extends StatelessWidget {
  final T value;
  final T groupValue;
  final ValueChanged<T> onChanged;
  final Widget child;
  final bool autofocus;

  const TVCardSelector({
    super.key,
    required this.value,
    required this.groupValue,
    required this.onChanged,
    required this.child,
    this.autofocus = false,
  });

  @override
  Widget build(BuildContext context) {
    final isSelected = value == groupValue;

    return TVFocusable(
      autofocus: autofocus,
      onTap: () => onChanged(value),
      borderRadius: BorderRadius.circular(12),
      focusBorderWidth: isSelected ? 3.0 : 2.0,
      focusBorderColor: isSelected
          ? Theme.of(context).colorScheme.primary
          : null,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        decoration: BoxDecoration(
          color: isSelected
              ? Theme.of(context).colorScheme.primaryContainer
              : null,
          borderRadius: BorderRadius.circular(12),
        ),
        child: child,
      ),
    );
  }
}
