package com.lord1egypt.pocketclaw.diagnostics

import java.io.File
import java.nio.file.Files
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

// PC-DEF-085. The lifecycle journal is the evidence for an unexplained start or
// crash after a reboot, so it must stay small, keep the newest lines, and never
// hold message text.
class LifecycleJournalTest {
    private lateinit var dir: File

    @Before
    fun setUp() {
        dir = Files.createTempDirectory("journal").toFile()
    }

    @After
    fun tearDown() {
        dir.deleteRecursively()
    }

    @Test
    fun `it never grows past its budget and keeps the newest lines`() {
        val journal = LifecycleJournal(File(dir, "diagnostics/lifecycle.log"), maxBytes = 2_000)
        repeat(500) { journal.append("entry-$it ${"x".repeat(40)}") }
        val text = journal.read()
        assertTrue(text.length <= 2_000)
        assertTrue(text.trimEnd().endsWith("entry-499 ${"x".repeat(40)}"))
        assertFalse(text.contains("entry-0 "))
    }

    @Test
    fun `one line stays one bounded line`() {
        val journal = LifecycleJournal(File(dir, "lifecycle.log"))
        journal.append("a\nb" + "y".repeat(1_000))
        val lines = journal.read().lines().filter { it.isNotEmpty() }
        assertEquals(1, lines.size)
        assertTrue(lines.single().length <= LifecycleJournal.MAX_LINE)
    }

    @Test
    fun `a missing journal reads as empty`() {
        assertEquals("", LifecycleJournal(File(dir, "absent/lifecycle.log")).read())
    }

    @Test
    fun `an exit description keeps the class and drops the message`() {
        assertEquals(
            "java.lang.IllegalStateException",
            LifecycleText.classOnly("java.lang.IllegalStateException: token sk-secret at /data/x"),
        )
        assertNull(LifecycleText.classOnly("user said: hello world"))
        assertNull(LifecycleText.classOnly(""))
        assertNull(LifecycleText.classOnly(null))
    }

    @Test
    fun `a crash summary names classes and our own frame, never the message`() {
        val error = IllegalStateException("password=hunter2", RuntimeException("inner secret"))
        error.stackTrace = arrayOf(
            StackTraceElement("android.app.Service", "startForeground", "Service.java", 1),
            StackTraceElement("com.lord1egypt.pocketclaw.service.PocketClawService", "onStartCommand", "S.kt", 2),
        )
        val summary = LifecycleText.crashSummary(error, "com.lord1egypt.pocketclaw")
        assertEquals(
            "error=java.lang.IllegalStateException at=PocketClawService.onStartCommand root=java.lang.RuntimeException",
            summary,
        )
        assertFalse(summary.contains("hunter2"))
        assertFalse(summary.contains("secret"))
    }

    @Test
    fun `an OS re-creation is recorded as a null intent`() {
        assertEquals("null-intent", LifecycleText.intentAction(false, null))
        assertEquals("no-action", LifecycleText.intentAction(true, null))
        assertEquals("com.lord1egypt.pocketclaw.action.START",
            LifecycleText.intentAction(true, "com.lord1egypt.pocketclaw.action.START"))
    }

    @Test
    fun `a line carries time, pid, component and event`() {
        val line = LifecycleText.line(0L, 42, "service", "start-command", "action=null-intent flags=0")
        assertEquals("1970-01-01T00:00:00.000Z | pid=42 | service | start-command | action=null-intent flags=0", line)
    }
}
