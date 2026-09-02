package com.lord1egypt.pocketclaw.config

import org.json.JSONArray
import org.json.JSONObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class ChannelListFieldTest {

    private val ownerPrincipal = "pico-user"

    /** Mirrors the provisioning decision in PicoClawService. */
    private fun isOwnerOnly(pico: JSONObject): Boolean {
        val allowFrom = ChannelListField.read(pico, "allow_from")
        return allowFrom.size == 1 && allowFrom[0] == ownerPrincipal
    }

    @Test
    fun `canonical array is read`() {
        val pico = JSONObject().put("allow_from", JSONArray().put("123").put("456"))
        assertEquals(listOf("123", "456"), ChannelListField.read(pico, "allow_from"))
    }

    @Test
    fun `legacy bare string is read as one entry`() {
        val pico = JSONObject().put("allow_from", ownerPrincipal)
        assertEquals(listOf(ownerPrincipal), ChannelListField.read(pico, "allow_from"))
    }

    @Test
    fun `legacy newline joined string is split`() {
        val pico = JSONObject().put("allow_from", "123\n456")
        assertEquals(listOf("123", "456"), ChannelListField.read(pico, "allow_from"))
    }

    @Test
    fun `blank lines and whitespace are dropped`() {
        val pico = JSONObject().put("allow_from", "  123  \n\n\r\n  456 \n   ")
        assertEquals(listOf("123", "456"), ChannelListField.read(pico, "allow_from"))
    }

    @Test
    fun `a stored entry containing a comma survives`() {
        val pico = JSONObject().put("allow_from", "a,b")
        assertEquals(listOf("a,b"), ChannelListField.read(pico, "allow_from"))
    }

    @Test
    fun `missing key reads as empty`() {
        assertEquals(emptyList<String>(), ChannelListField.read(JSONObject(), "allow_from"))
    }

    @Test
    fun `unsupported types read as empty rather than throwing`() {
        assertEquals(
            emptyList<String>(),
            ChannelListField.read(JSONObject().put("allow_from", 42), "allow_from"),
        )
        assertEquals(
            emptyList<String>(),
            ChannelListField.read(
                JSONObject().put("allow_from", JSONObject().put("nested", "x")),
                "allow_from",
            ),
        )
    }

    // The regression this fixes: a valid legacy config must not look
    // unprovisioned, or the service rewrites config.json on every start.
    @Test
    fun `legacy owner-only string does not trigger reprovision`() {
        assertTrue(isOwnerOnly(JSONObject().put("allow_from", ownerPrincipal)))
    }

    @Test
    fun `canonical owner-only array does not trigger reprovision`() {
        assertTrue(
            isOwnerOnly(JSONObject().put("allow_from", JSONArray().put(ownerPrincipal))),
        )
    }

    @Test
    fun `owner check is not weakened by the legacy shape`() {
        // More than the owner, in either shape, is still not owner-only.
        assertFalse(isOwnerOnly(JSONObject().put("allow_from", "$ownerPrincipal\nintruder")))
        assertFalse(
            isOwnerOnly(
                JSONObject().put(
                    "allow_from",
                    JSONArray().put(ownerPrincipal).put("intruder"),
                ),
            ),
        )
        // A different principal is not the owner.
        assertFalse(isOwnerOnly(JSONObject().put("allow_from", "someone-else")))
        // Absent allow_from is not owner-only.
        assertFalse(isOwnerOnly(JSONObject()))
    }
}
