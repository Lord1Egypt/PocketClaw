"""PocketClaw's Python entry point.

Android's CPython does not connect sys.stdout and sys.stderr to file
descriptors 1 and 2. It replaces both with TextLogStream, which writes to the
Android system log, because an app has no console. The managed runtime captures
the descriptors, so on a device every print() went to logcat and the caller saw
an empty stream: a silent no-op that still exited 0, and an uncaught exception
whose traceback was never shown even though the exit status was 1.

This module runs in the caller's place. It restores the two streams, reads the
program from standard input exactly as `python -` does, and executes it. It is
PocketClaw code shipped inside the checksum-verified payload; the program it
runs is the only thing that comes from the caller, and it arrives on stdin.
"""

import builtins
import io
import os
import sys
import traceback
import types

# The filename the caller's code is compiled under. It is what appears in a
# traceback, and it stays `<stdin>` because that is where the program came from
# and what `python -` would have reported.
PROGRAM_NAME = "<stdin>"

# What `python -` puts in sys.argv[0]. The traceback name and the argv name are
# different strings in CPython and both are part of the contract, so neither is
# derived from the other.
ARGV0 = "-"

# Distinct from any status the caller's program can produce, so a failure to
# start is never mistaken for a failure of the program.
BOOTSTRAP_FAILURE = 70


def _abort(message):
    """Report a failure that happened before the streams could be trusted.

    Writing to the descriptor directly is the only reliable channel here: the
    whole reason this module exists is that sys.stderr may go somewhere the
    caller cannot see.
    """
    try:
        os.write(2, ("pocketclaw-bootstrap: " + message + "\n").encode(
            "utf-8", "backslashreplace"))
    except OSError:
        pass
    os._exit(BOOTSTRAP_FAILURE)


def _descriptor_stream(fd, errors):
    """Bind a text stream to a file descriptor, unbuffered.

    The descriptor is duplicated so that closing this wrapper at interpreter
    shutdown closes the duplicate rather than the descriptor the runtime is
    reading from. Writes go straight through, which is what the runtime's -u
    asks for: output must be visible even if the process is terminated.
    """
    duplicate = os.dup(fd)
    raw = io.FileIO(duplicate, "wb", closefd=True)
    return io.TextIOWrapper(
        raw, encoding="utf-8", errors=errors, write_through=True)


def _restore_streams():
    try:
        sys.stdout = sys.__stdout__ = _descriptor_stream(1, "strict")
        sys.stderr = sys.__stderr__ = _descriptor_stream(2, "backslashreplace")
    except OSError as error:
        _abort("cannot bind standard output to its file descriptor: %s" % error)


def _read_program():
    """Read the whole of standard input as the program to run.

    Phase C has no second input channel: stdin carries the source and nothing
    else, so consuming it entirely is the contract rather than a shortcut. A
    program that needs data embeds it or reads a workspace file.
    """
    try:
        stream = getattr(sys.stdin, "buffer", None)
        if stream is not None:
            raw = stream.read()
        else:
            chunks = []
            while True:
                chunk = os.read(0, 65536)
                if not chunk:
                    break
                chunks.append(chunk)
            raw = b"".join(chunks)
    except OSError as error:
        _abort("cannot read the program from standard input: %s" % error)

    try:
        return raw.decode("utf-8")
    except UnicodeDecodeError as error:
        _abort("the program is not valid UTF-8: %s" % error)


def _exit_status(request):
    code = request.code
    if code is None:
        return 0
    if isinstance(code, int):
        return code
    print(code, file=sys.stderr)
    return 1


def _report(error):
    """Print the caller's traceback, without this module's frames in it.

    The first frame belongs to the exec() below. Dropping it leaves a traceback
    whose filenames and line numbers refer only to the code the caller sent,
    which is what the line numbers in an error message have to mean.
    """
    trace = error.__traceback__
    traceback.print_exception(type(error), error,
                              trace.tb_next if trace is not None else None)


def main():
    _restore_streams()
    source = _read_program()

    # `python -` reports the program as argv[0]; keep that contract so a script
    # reading sys.argv sees what it would outside PocketClaw, and so the
    # bootstrap's own module path never appears.
    sys.argv = [ARGV0] + sys.argv[1:]

    program = types.ModuleType("__main__")
    program.__dict__["__builtins__"] = builtins
    sys.modules["__main__"] = program

    try:
        exec(compile(source, PROGRAM_NAME, "exec"), program.__dict__)
    except SystemExit as request:
        status = _exit_status(request)
    except BaseException as error:
        _report(error)
        status = 1
    else:
        status = 0

    for stream in (sys.stdout, sys.stderr):
        try:
            stream.flush()
        except (OSError, ValueError):
            pass
    return status


if __name__ == "__main__":
    sys.exit(main())
