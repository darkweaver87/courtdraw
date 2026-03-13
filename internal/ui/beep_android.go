//go:build android

package ui

/*
#cgo LDFLAGS: -lm -ldl
#include <dlfcn.h>
#include <math.h>
#include <stdlib.h>
#include <stdint.h>
#include <unistd.h>

typedef int32_t (*pfn_create_builder)(void**);
typedef void    (*pfn_set_i32)(void*, int32_t);
typedef int32_t (*pfn_open_stream)(void*, void**);
typedef int32_t (*pfn_op)(void*);
typedef int32_t (*pfn_write)(void*, const void*, int32_t, int64_t);

static void play_beep() {
	void *lib = dlopen("libaaudio.so", RTLD_LAZY);
	if (!lib) return;

	pfn_create_builder createBuilder = dlsym(lib, "AAudio_createStreamBuilder");
	pfn_set_i32 setSR   = dlsym(lib, "AAudioStreamBuilder_setSampleRate");
	pfn_set_i32 setCh   = dlsym(lib, "AAudioStreamBuilder_setChannelCount");
	pfn_set_i32 setFmt  = dlsym(lib, "AAudioStreamBuilder_setFormat");
	pfn_set_i32 setDir  = dlsym(lib, "AAudioStreamBuilder_setDirection");
	pfn_open_stream openStream = dlsym(lib, "AAudioStreamBuilder_openStream");
	pfn_op delBuilder = dlsym(lib, "AAudioStreamBuilder_delete");
	pfn_op reqStart   = dlsym(lib, "AAudioStream_requestStart");
	pfn_write write   = dlsym(lib, "AAudioStream_write");
	pfn_op reqStop    = dlsym(lib, "AAudioStream_requestStop");
	pfn_op close      = dlsym(lib, "AAudioStream_close");

	if (!createBuilder || !openStream) { dlclose(lib); return; }

	void *builder = NULL;
	if (createBuilder(&builder) != 0) { dlclose(lib); return; }

	setSR(builder, 44100);
	setCh(builder, 1);
	setFmt(builder, 1);  // AAUDIO_FORMAT_PCM_I16
	setDir(builder, 0);  // AAUDIO_DIRECTION_OUTPUT

	void *stream = NULL;
	if (openStream(builder, &stream) != 0) { delBuilder(builder); dlclose(lib); return; }
	delBuilder(builder);

	// 200ms of 880 Hz sine wave
	const int sr = 44100, n = 44100 * 200 / 1000, fade = 44100 * 10 / 1000;
	int16_t *buf = malloc(n * sizeof(int16_t));
	for (int i = 0; i < n; i++) {
		double env = 1.0;
		if (i < fade) env = (double)i / fade;
		else if (n - i < fade) env = (double)(n - i) / fade;
		buf[i] = (int16_t)(env * 0.4 * sin(2.0 * M_PI * 880.0 * i / sr) * 32767);
	}

	reqStart(stream);
	write(stream, buf, n, 1000000000LL);
	usleep(250000);
	reqStop(stream);
	close(stream);
	free(buf);
	dlclose(lib);
}
*/
import "C"

func systemBeep() {
	go func() {
		C.play_beep()
	}()
}
