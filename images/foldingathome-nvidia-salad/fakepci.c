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
  if (flags & (O_CREAT | O_TMPFILE)) { va_list ap; va_start(ap, flags); mode = va_arg(ap, mode_t); va_end(ap); }
  return real(redirect(path, b, sizeof(b)), flags, mode);
}

int open64(const char *path, int flags, ...) {
  char b[4096]; REAL(open64);
  mode_t mode = 0;
  if (flags & (O_CREAT | O_TMPFILE)) { va_list ap; va_start(ap, flags); mode = va_arg(ap, mode_t); va_end(ap); }
  return real(redirect(path, b, sizeof(b)), flags, mode);
}
