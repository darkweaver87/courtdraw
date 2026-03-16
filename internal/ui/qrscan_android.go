//go:build android

package ui

/*
#cgo LDFLAGS: -ldl
#include <jni.h>
#include <dlfcn.h>
#include <stdlib.h>
#include <string.h>

typedef jint (*pfn_get_created_vms)(JavaVM**, jsize, jsize*);

static JavaVM *g_jvm = NULL;
static jobject g_photo_uri = NULL; // global ref to content URI

static JavaVM* get_jvm() {
	if (g_jvm) return g_jvm;
	void *lib = dlopen("libnativehelper.so", RTLD_LAZY);
	if (!lib) lib = dlopen("libart.so", RTLD_LAZY);
	if (!lib) return NULL;
	pfn_get_created_vms fn = (pfn_get_created_vms)dlsym(lib, "JNI_GetCreatedJavaVMs");
	if (!fn) { dlclose(lib); return NULL; }
	JavaVM *vms[1]; jsize count = 0;
	if (fn(vms, 1, &count) != 0 || count == 0) { dlclose(lib); return NULL; }
	g_jvm = vms[0];
	dlclose(lib);
	return g_jvm;
}

static JNIEnv* get_env(int *attached) {
	JavaVM *jvm = get_jvm();
	if (!jvm) return NULL;
	JNIEnv *env = NULL;
	*attached = 0;
	if ((*jvm)->GetEnv(jvm, (void**)&env, JNI_VERSION_1_6) != JNI_OK) {
		if ((*jvm)->AttachCurrentThread(jvm, &env, NULL) != 0) return NULL;
		*attached = 1;
	}
	return env;
}

static jobject get_context(JNIEnv *env) {
	jclass atClass = (*env)->FindClass(env, "android/app/ActivityThread");
	if (!atClass || (*env)->ExceptionCheck(env)) return NULL;
	jmethodID currentAT = (*env)->GetStaticMethodID(env, atClass,
		"currentActivityThread", "()Landroid/app/ActivityThread;");
	jobject at = (*env)->CallStaticObjectMethod(env, atClass, currentAT);
	if (!at) return NULL;
	jmethodID getApp = (*env)->GetMethodID(env, atClass,
		"getApplication", "()Landroid/app/Application;");
	return (*env)->CallObjectMethod(env, at, getApp);
}

// start_qr_capture creates a MediaStore entry, launches the camera to save
// a photo there, and stores the URI for later reading. Returns 0 on success.
static int start_qr_capture() {
	int attached;
	JNIEnv *env = get_env(&attached);
	if (!env) return -1;

	jobject ctx = get_context(env);
	if (!ctx) goto fail;

	// ContentValues values = new ContentValues();
	jclass cvClass = (*env)->FindClass(env, "android/content/ContentValues");
	jmethodID cvInit = (*env)->GetMethodID(env, cvClass, "<init>", "()V");
	jobject values = (*env)->NewObject(env, cvClass, cvInit);

	// values.put("_display_name", "courtdraw_qr.jpg");
	// values.put("mime_type", "image/jpeg");
	jmethodID putStr = (*env)->GetMethodID(env, cvClass, "put",
		"(Ljava/lang/String;Ljava/lang/String;)V");
	(*env)->CallVoidMethod(env, values, putStr,
		(*env)->NewStringUTF(env, "_display_name"),
		(*env)->NewStringUTF(env, "courtdraw_qr.jpg"));
	(*env)->CallVoidMethod(env, values, putStr,
		(*env)->NewStringUTF(env, "mime_type"),
		(*env)->NewStringUTF(env, "image/jpeg"));

	// Uri uri = cr.insert(MediaStore.Images.Media.EXTERNAL_CONTENT_URI, values);
	jmethodID getCR = (*env)->GetMethodID(env,
		(*env)->GetObjectClass(env, ctx),
		"getContentResolver", "()Landroid/content/ContentResolver;");
	jobject cr = (*env)->CallObjectMethod(env, ctx, getCR);

	jclass msClass = (*env)->FindClass(env, "android/provider/MediaStore$Images$Media");
	jfieldID extUriField = (*env)->GetStaticFieldID(env, msClass,
		"EXTERNAL_CONTENT_URI", "Landroid/net/Uri;");
	jobject extUri = (*env)->GetStaticObjectField(env, msClass, extUriField);

	jclass crClass = (*env)->FindClass(env, "android/content/ContentResolver");
	jmethodID insertMethod = (*env)->GetMethodID(env, crClass, "insert",
		"(Landroid/net/Uri;Landroid/content/ContentValues;)Landroid/net/Uri;");
	jobject photoUri = (*env)->CallObjectMethod(env, cr, insertMethod, extUri, values);
	if (!photoUri || (*env)->ExceptionCheck(env)) goto fail;

	// Store URI globally.
	if (g_photo_uri) (*env)->DeleteGlobalRef(env, g_photo_uri);
	g_photo_uri = (*env)->NewGlobalRef(env, photoUri);

	// Intent intent = new Intent(MediaStore.ACTION_IMAGE_CAPTURE);
	jclass intentClass = (*env)->FindClass(env, "android/content/Intent");
	jstring action = (*env)->NewStringUTF(env, "android.media.action.IMAGE_CAPTURE");
	jmethodID intentInit = (*env)->GetMethodID(env, intentClass,
		"<init>", "(Ljava/lang/String;)V");
	jobject intent = (*env)->NewObject(env, intentClass, intentInit, action);

	// intent.putExtra(MediaStore.EXTRA_OUTPUT, photoUri);
	jstring extraKey = (*env)->NewStringUTF(env, "output");
	jmethodID putExtra = (*env)->GetMethodID(env, intentClass, "putExtra",
		"(Ljava/lang/String;Landroid/os/Parcelable;)Landroid/content/Intent;");
	(*env)->CallObjectMethod(env, intent, putExtra, extraKey, photoUri);

	// intent.addFlags(FLAG_ACTIVITY_NEW_TASK | FLAG_GRANT_WRITE_URI_PERMISSION);
	jmethodID addFlags = (*env)->GetMethodID(env, intentClass, "addFlags",
		"(I)Landroid/content/Intent;");
	(*env)->CallObjectMethod(env, intent, addFlags, 0x10000000 | 0x00000002);

	// ctx.startActivity(intent);
	jmethodID startActivity = (*env)->GetMethodID(env,
		(*env)->GetObjectClass(env, ctx), "startActivity",
		"(Landroid/content/Intent;)V");
	(*env)->CallVoidMethod(env, ctx, startActivity, intent);

	if ((*env)->ExceptionCheck(env)) goto fail;
	if (attached) (*get_jvm())->DetachCurrentThread(get_jvm());
	return 0;

fail:
	if ((*env)->ExceptionCheck(env)) (*env)->ExceptionClear(env);
	if (attached) (*get_jvm())->DetachCurrentThread(get_jvm());
	return -1;
}

// read_captured_photo reads the photo bytes from the stored content URI.
// On success, sets *out_data (caller must free) and returns the byte count.
// On failure, returns -1.
static int read_captured_photo(unsigned char **out_data) {
	if (!g_photo_uri) return -1;

	int attached;
	JNIEnv *env = get_env(&attached);
	if (!env) return -1;

	jobject ctx = get_context(env);
	if (!ctx) goto fail;

	jmethodID getCR = (*env)->GetMethodID(env,
		(*env)->GetObjectClass(env, ctx),
		"getContentResolver", "()Landroid/content/ContentResolver;");
	jobject cr = (*env)->CallObjectMethod(env, ctx, getCR);

	// InputStream is = cr.openInputStream(uri);
	jclass crClass = (*env)->FindClass(env, "android/content/ContentResolver");
	jmethodID openIS = (*env)->GetMethodID(env, crClass, "openInputStream",
		"(Landroid/net/Uri;)Ljava/io/InputStream;");
	jobject is = (*env)->CallObjectMethod(env, cr, openIS, g_photo_uri);
	if (!is || (*env)->ExceptionCheck(env)) goto fail;

	// Read into ByteArrayOutputStream.
	jclass baosClass = (*env)->FindClass(env, "java/io/ByteArrayOutputStream");
	jmethodID baosInit = (*env)->GetMethodID(env, baosClass, "<init>", "()V");
	jobject baos = (*env)->NewObject(env, baosClass, baosInit);

	jbyteArray buf = (*env)->NewByteArray(env, 16384);
	jclass isClass = (*env)->FindClass(env, "java/io/InputStream");
	jmethodID readM = (*env)->GetMethodID(env, isClass, "read", "([B)I");
	jmethodID writeM = (*env)->GetMethodID(env, baosClass, "write", "([BII)V");

	while (1) {
		jint n = (*env)->CallIntMethod(env, is, readM, buf);
		if (n <= 0) break;
		(*env)->CallVoidMethod(env, baos, writeM, buf, 0, n);
	}

	// Close stream.
	jmethodID closeM = (*env)->GetMethodID(env, isClass, "close", "()V");
	(*env)->CallVoidMethod(env, is, closeM);

	// Get result bytes.
	jmethodID toBytes = (*env)->GetMethodID(env, baosClass, "toByteArray", "()[B");
	jbyteArray data = (*env)->CallObjectMethod(env, baos, toBytes);
	jint len = (*env)->GetArrayLength(env, data);

	if (len <= 0) goto cleanup;

	*out_data = (unsigned char*)malloc(len);
	(*env)->GetByteArrayRegion(env, data, 0, len, (jbyte*)*out_data);

	// Delete the MediaStore entry (cleanup).
	jmethodID delM = (*env)->GetMethodID(env, crClass, "delete",
		"(Landroid/net/Uri;Ljava/lang/String;[Ljava/lang/String;)I");
	(*env)->CallIntMethod(env, cr, delM, g_photo_uri, NULL, NULL);

cleanup:
	(*env)->DeleteGlobalRef(env, g_photo_uri);
	g_photo_uri = NULL;
	if (attached) (*get_jvm())->DetachCurrentThread(get_jvm());
	return (int)len;

fail:
	if ((*env)->ExceptionCheck(env)) (*env)->ExceptionClear(env);
	if (g_photo_uri) {
		(*env)->DeleteGlobalRef(env, g_photo_uri);
		g_photo_uri = NULL;
	}
	if (attached) (*get_jvm())->DetachCurrentThread(get_jvm());
	return -1;
}

// cleanup_photo_uri releases the global URI ref if the user cancelled.
static void cleanup_photo_uri() {
	if (!g_photo_uri) return;
	int attached;
	JNIEnv *env = get_env(&attached);
	if (!env) return;

	// Try to delete the empty MediaStore entry.
	jobject ctx = get_context(env);
	if (ctx) {
		jmethodID getCR = (*env)->GetMethodID(env,
			(*env)->GetObjectClass(env, ctx),
			"getContentResolver", "()Landroid/content/ContentResolver;");
		jobject cr = (*env)->CallObjectMethod(env, ctx, getCR);
		jclass crClass = (*env)->FindClass(env, "android/content/ContentResolver");
		jmethodID delM = (*env)->GetMethodID(env, crClass, "delete",
			"(Landroid/net/Uri;Ljava/lang/String;[Ljava/lang/String;)I");
		(*env)->CallIntMethod(env, cr, delM, g_photo_uri, NULL, NULL);
	}
	if ((*env)->ExceptionCheck(env)) (*env)->ExceptionClear(env);

	(*env)->DeleteGlobalRef(env, g_photo_uri);
	g_photo_uri = NULL;
	if (attached) (*get_jvm())->DetachCurrentThread(get_jvm());
}
*/
import "C"

import "unsafe"

// openCameraForQR launches the camera to take a photo (saved via MediaStore).
func openCameraForQR() bool {
	return C.start_qr_capture() == 0
}

// readCapturedPhoto reads the photo bytes taken by the camera.
// Returns nil if no photo is available (user cancelled or error).
func readCapturedPhoto() []byte {
	var cData *C.uchar
	n := C.read_captured_photo(&cData)
	if n <= 0 || cData == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(cData))
	return C.GoBytes(unsafe.Pointer(cData), n)
}

// cleanupPhotoURI releases the MediaStore entry if the scan was cancelled.
func cleanupPhotoURI() {
	C.cleanup_photo_uri()
}
