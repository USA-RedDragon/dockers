// LD_PRELOAD shim that redirects reads of /proc/bus/pci/devices to $FAKE_PCI_PROC
// (what fah-client 8.5.6 reads) and of /sys/bus/pci/devices to $FAKE_PCI_ROOT (newer cbang).
// Under WSL2 the GPU is not on a visible PCI bus, so fah-client cannot learn its
// PCI device ID and never marks it supported. salad-start writes the real IDs,
// taken from nvidia-smi, into $FAKE_PCI_ROOT before starting the client.
#define _GNU_SOURCE
#include <dirent.h>
#include <dlfcn.h>
#include <fcntl.h>
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>

static const char prefix[] = "/sys/bus/pci/devices";

static const char *redirect(const char *path, char *buf, size_t len) {
  const char *proc = getenv("FAKE_PCI_PROC");
  if (proc && path && !strcmp(path, "/proc/bus/pci/devices")) return proc;
  const char *root = getenv("FAKE_PCI_ROOT");
  size_t n = sizeof(prefix) - 1;
  if (!root || !path || strncmp(path, prefix, n) || (path[n] && path[n] != '/'))
    return path;
  snprintf(buf, len, "%s%s", root, path + n);
  return buf;
}

#define REAL(name) static __typeof__(name) *real; if (!real) real = dlsym(RTLD_NEXT, #name)

DIR *opendir(const char *path) {
  char b[4096]; REAL(opendir);
  return real(redirect(path, b, sizeof(b)));
}

int stat(const char *path, struct stat *st) {
  char b[4096]; REAL(stat);
  return real(redirect(path, b, sizeof(b)), st);
}

int lstat(const char *path, struct stat *st) {
  char b[4096]; REAL(lstat);
  return real(redirect(path, b, sizeof(b)), st);
}

int stat64(const char *path, struct stat64 *st) {
  char b[4096]; REAL(stat64);
  return real(redirect(path, b, sizeof(b)), st);
}

int access(const char *path, int mode) {
  char b[4096]; REAL(access);
  return real(redirect(path, b, sizeof(b)), mode);
}

FILE *fopen(const char *path, const char *mode) {
  char b[4096]; REAL(fopen);
  return real(redirect(path, b, sizeof(b)), mode);
}

FILE *fopen64(const char *path, const char *mode) {
  char b[4096]; REAL(fopen64);
  return real(redirect(path, b, sizeof(b)), mode);
}

int open(const char *path, int flags, ...) {
  char b[4096]; REAL(open);
  mode_t mode = 0;
  if ((flags & O_CREAT) || (flags & O_TMPFILE) == O_TMPFILE) { va_list ap; va_start(ap, flags); mode = va_arg(ap, mode_t); va_end(ap); }
  return real(redirect(path, b, sizeof(b)), flags, mode);
}

// fah-client names a GPU after the PCI bus and slot that CUDA and OpenCL report, and that name is written into the
// signed assignment of every unit it gets. Salad nodes put the GPU at different addresses, so inside fah-client the
// address is replaced with $FAKE_GPU_BUS/$FAKE_GPU_SLOT (decimal). cbang looks these functions up with dlsym.
typedef void *(*dlsym_t)(void *, const char *);
typedef int (*cu_attr_t)(int *, int, int);
typedef int (*cl_info_t)(void *, unsigned, size_t, void *, size_t *);
static cu_attr_t real_cu_attr;
static cl_info_t real_cl_info;

static dlsym_t real_dlsym(void) {
  static dlsym_t f;
  if (!f) f = (dlsym_t)dlvsym(RTLD_NEXT, "dlsym", "GLIBC_2.34");
  if (!f) f = (dlsym_t)dlvsym(RTLD_NEXT, "dlsym", "GLIBC_2.2.5");
  return f;
}

static int fake_addr(int *bus, int *slot) {
  extern char *program_invocation_short_name;
  const char *b = getenv("FAKE_GPU_BUS"), *s = getenv("FAKE_GPU_SLOT");
  if (!b || strcmp(program_invocation_short_name, "fah-client")) return 0;
  *bus = atoi(b);
  *slot = s ? atoi(s) : 0;
  return 1;
}

static int fake_cu_attr(int *value, int attr, int dev) {
  int r = real_cu_attr(value, attr, dev), bus, slot;
  if (!r && value && fake_addr(&bus, &slot)) {
    if (attr == 33) *value = bus;
    else if (attr == 34) *value = slot;
    else if (attr == 50) *value = 0;
  }
  return r;
}

static int fake_cl_info(void *dev, unsigned param, size_t size, void *value, size_t *ret) {
  int r = real_cl_info(dev, param, size, value, ret), bus, slot;
  if (!r && value && fake_addr(&bus, &slot)) {
    unsigned *v = value;
    if (param == 0x4008 && size >= 4) v[0] = bus;
    else if (param == 0x4009 && size >= 4) v[0] = slot;
    else if (param == 0x410f && size >= 16) { v[0] = 0; v[1] = bus; v[2] = slot; v[3] = 0; }
  }
  return r;
}

// glibc resolves RTLD_NEXT (and scopes RTLD_DEFAULT) relative to the caller's return address, so every lookup that is
// not one of the two faked functions must reach glibc as a tail call; otherwise a wrapper library's
// dlsym(RTLD_NEXT, x) would return the wrapper itself and recurse (this crashed fah-client under WSL's libcuda).
static void *fake_lookup(void *handle, const char *name, dlsym_t real) __attribute__((noinline));
static void *fake_lookup(void *handle, const char *name, dlsym_t real) {
  void *p = real(handle, name);
  if (!p) return p;
  if (name[0] == 'c' && !strcmp(name, "cuDeviceGetAttribute")) { real_cu_attr = (cu_attr_t)p; return (void *)fake_cu_attr; }
  real_cl_info = (cl_info_t)p;
  return (void *)fake_cl_info;
}

void *dlsym(void *handle, const char *name) {
  dlsym_t real = real_dlsym();
  int bus, slot;
  if (handle != RTLD_NEXT && handle != RTLD_DEFAULT &&
      (!strcmp(name, "cuDeviceGetAttribute") || !strcmp(name, "clGetDeviceInfo")) && fake_addr(&bus, &slot))
    return fake_lookup(handle, name, real);
  return real(handle, name);
}

int open64(const char *path, int flags, ...) {
  char b[4096]; REAL(open64);
  mode_t mode = 0;
  if ((flags & O_CREAT) || (flags & O_TMPFILE) == O_TMPFILE) { va_list ap; va_start(ap, flags); mode = va_arg(ap, mode_t); va_end(ap); }
  return real(redirect(path, b, sizeof(b)), flags, mode);
}
