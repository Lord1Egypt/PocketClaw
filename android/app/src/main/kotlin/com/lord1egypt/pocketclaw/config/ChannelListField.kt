package com.lord1egypt.pocketclaw.config

import org.json.JSONObject

/**
 * 读取 channel config 中的列表字段。
 *
 * Canonically these are JSON arrays. Older console builds wrote them as a
 * single newline-joined string and those configs are still on disk, so both
 * shapes must read the same way here: `optJSONArray` returns null for the
 * legacy string, which made provisioning treat a perfectly valid value as
 * missing and rewrite config.json on every service start.
 *
 * Reading is deliberately non-destructive — a legacy value is understood, not
 * rewritten. The console repairs the shape the next time the user saves.
 */
object ChannelListField {

    /** Reads [key] from [owner] as a list, accepting either shape. */
    fun read(owner: JSONObject, key: String): List<String> {
        val array = owner.optJSONArray(key)
        if (array != null) {
            return normalize((0 until array.length()).map { array.optString(it) })
        }
        val legacy = owner.opt(key)
        if (legacy is String) {
            return normalize(splitLegacy(legacy))
        }
        return emptyList()
    }

    /**
     * Splits the legacy shape. The only separator a stored value ever had is
     * the "\n" the old console joined on, so an entry containing a comma or a
     * semicolon survives intact.
     */
    fun splitLegacy(raw: String): List<String> = raw.split('\n')

    /** Trims entries and drops blanks, leaving order and duplicates alone. */
    fun normalize(items: List<String>): List<String> =
        items.map { it.trim() }.filter { it.isNotEmpty() }
}
