#ifndef POCKETCLAW_ANDROID_PTHREAD_CANCEL_H
#define POCKETCLAW_ANDROID_PTHREAD_CANCEL_H

/*
 * Android's bionic deliberately implements no pthread cancellation, so it
 * declares neither pthread_setcancelstate() nor PTHREAD_CANCEL_*.
 *
 * git calls pthread_setcancelstate() only to make an async section
 * non-cancellable and then restore the previous state. On a platform where no
 * thread can ever be cancelled, "make this uncancellable" is already true, so a
 * no-op that reports success is the correct behaviour rather than a stub that
 * papers over missing functionality.
 */
#define PTHREAD_CANCEL_ENABLE 0
#define PTHREAD_CANCEL_DISABLE 1

static inline int pthread_setcancelstate(int state, int *oldstate)
{
	(void)state;
	if (oldstate)
		*oldstate = PTHREAD_CANCEL_ENABLE;
	return 0;
}

#endif /* POCKETCLAW_ANDROID_PTHREAD_CANCEL_H */
