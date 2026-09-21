# PocketClaw application R8 rules
#
# Deliberately no application-wide keep rule lives here. Android's generated
# aapt rules retain manifest-instantiated components, Flutter 3.47.1 supplies
# its embedding/plugin contract, and each plugin supplies its consumer rules.
# Keeping com.lord1egypt.pocketclaw.**, io.flutter.**, or every plugin class
# would preserve implementation names and defeat release obfuscation.
#
# Add a rule only when a concrete reflection/JNI/framework entry point cannot
# be represented by the manifest, an annotation, or the owning dependency's
# consumer rules. Every addition must have a focused regression test.
