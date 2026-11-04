// LD_PRELOAD shim that redirects reads of /proc/bus/pci/devices to $FAKE_PCI_PROC
// (what fah-client 8.5.6 reads) and of /sys/bus/pci/devices to $FAKE_PCI_ROOT (newer cbang).
// Under WSL2 the GPU is not on a visible PCI bus, so fah-client cannot learn its
// PCI device ID and never marks it supported. salad-start writes the real IDs,
// taken from nvidia-smi, into $FAKE_PCI_ROOT before starting the client.
#define _GNU_SOURCE
#include <dirent.h>
#include <dlfcn.h>
#include <fcntl.h>
#include <link.h>
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <unistd.h>

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
// Under WSL a later lookup can hand back the wrapper itself; storing that as the real function made fake_cu_attr call
// itself until the stack overflowed. Only a genuine function is ever stored, and the first one is kept.
static void *fake_lookup(void *handle, const char *name, dlsym_t real) {
  void *p = real(handle, name);
  if (!p) return p;
  if (!strcmp(name, "cuDeviceGetAttribute")) {
    if (p != (void *)fake_cu_attr && !real_cu_attr) real_cu_attr = (cu_attr_t)p;
    return real_cu_attr ? (void *)fake_cu_attr : p;
  }
  if (p != (void *)fake_cl_info && !real_cl_info) real_cl_info = (cl_info_t)p;
  return real_cl_info ? (void *)fake_cl_info : p;
}

// Only lookups made by the fah-client executable itself (cbang is linked into it) get the wrapper. WSL's libcuda
// forwards cuDeviceGetAttribute by looking the same name up in its backend with dlsym; handing it the wrapper made the
// wrapper and libcuda call each other until the stack overflowed.
static int from_executable(void *addr) {
  Dl_info info;
  struct link_map *map = 0;
  if (!dladdr1(addr, &info, (void **)&map, RTLD_DL_LINKMAP) || !map || !map->l_name) return 0;
  return map->l_name[0] == '\0';
}

void *dlsym(void *handle, const char *name) {
  dlsym_t real = real_dlsym();
  int bus, slot;
  if (handle != RTLD_NEXT && handle != RTLD_DEFAULT &&
      (!strcmp(name, "cuDeviceGetAttribute") || !strcmp(name, "clGetDeviceInfo")) && fake_addr(&bus, &slot) &&
      from_executable(__builtin_return_address(0)))
    return fake_lookup(handle, name, real);
  return real(handle, name);
}

// An OpenMM 8 CUDA checkpoint holds one random seed per GPU thread block (numThreadBlocks: 6 per SM, 4 on Pascal), and
// OpenMM reads it back with the reading GPU's count. Resumed on a GPU with fewer SMs, the rest of the file is read out
// of step, the CPU random generator gets a garbage index, and the core crashes with SIGSEGV about half the time (the
// unit is dumped). The checkpoint is integrity checked, so it cannot be fixed; instead a FahCore that resumes one is told
// the SM count that gives the block count the checkpoint was written with. Only more SMs are ever reported: telling a
// 4090 it had 66 SMs was followed by CUDA_ERROR_ILLEGAL_ADDRESS 145 steps after the resume.
static long checkpoint_blocks(void) {
  char args[4096], path[4200], *dir = 0, *suffix = 0;
  FILE *f = fopen("/proc/self/cmdline", "r");
  if (!f) return 0;
  size_t n = fread(args, 1, sizeof(args) - 1, f);
  fclose(f);
  args[n] = 0;
  for (char *a = args; a < args + n; a += strlen(a) + 1) {
    char *v = a + strlen(a) + 1;
    if (v >= args + n) break;
    if (!strcmp(a, "-dir")) dir = v;
    else if (!strcmp(a, "-suffix")) suffix = v;
  }
  if (!dir || !suffix || strchr(dir, '/') || strchr(suffix, '/')) return 0;
  // The core runs from the work folder and then changes into <dir>/<suffix>.
  char cwd[4096], tail[4200];
  snprintf(path, sizeof(path), "%s/%s/checkpoint", dir, suffix);
  snprintf(tail, sizeof(tail), "/%s/%s", dir, suffix);
  size_t cl = getcwd(cwd, sizeof(cwd)) ? strlen(cwd) : 0, tl = strlen(tail);
  if (cl >= tl && !strcmp(cwd + cl - tl, tail)) snprintf(path, sizeof(path), "checkpoint");
  int fd = open(path, O_RDONLY);
  if (fd < 0) return 0;
  struct stat st;
  long blocks = 0;
  unsigned char h[4096];
  ssize_t got = fstat(fd, &st) ? -1 : pread(fd, h, sizeof(h), 0);
  static const char magic[] = "OpenMM Binary Checkpoint\n";
  size_t o = sizeof(magic);
  int i32;
#define TAKE(dst, len) do { if (o + (len) > (size_t)got) goto out; memcpy((dst), h + o, (len)); o += (len); } while (0)
  if (got < (ssize_t)sizeof(magic) || memcmp(h, magic, sizeof(magic))) goto out;
  TAKE(&i32, 4);
  if (i32 != 4 || o + 4 > (size_t)got || memcmp(h + o, "CUDA", 4)) goto out;
  o += 4;
  int atoms, params, version, precision;
  TAKE(&atoms, 4);
  TAKE(&params, 4);
  if (atoms <= 0 || params < 0 || params > 100) goto out;
  for (int p = 0; p < params; p++) {
    TAKE(&i32, 4);
    if (i32 < 0 || i32 > 1024) goto out;
    o += i32 + 8;
  }
  TAKE(&version, 4);
  TAKE(&precision, 4);
  if (version != 3 || precision < 0 || precision > 2) goto out;
  o += 8 + 8 + 4;
  long long padded = 32LL * ((atoms + 31) / 32);
  static const int bytes[3][3] = {{16, 0, 16}, {16, 16, 32}, {32, 0, 32}};
  long long rng = o + padded * (bytes[precision][0] + bytes[precision][1] + bytes[precision][2] + 4 + 16) + 72;
  long long seeds = rng + 4 + padded * 4 * 16;
  static const int tails[2] = {4 + 1 + 1 + 8 + 156 * 16 + 4, 4 + 1};
  for (int t = 0; t < 2; t++) {
    long long left = (long long)st.st_size - tails[t] - seeds;
    unsigned char init;
    if (left <= 0 || left % 1024) continue;
    if (pread(fd, &init, 1, st.st_size - tails[t] + 4) != 1 || (t == 0) != (init != 0)) continue;
    blocks = left / 1024;
    break;
  }
#undef TAKE
out:
  close(fd);
  return blocks;
}

int cuDeviceGetAttribute(int *value, int attr, int dev) {
  extern char *program_invocation_short_name;
  static cu_attr_t next;
  static long blocks = -1;
  static __thread int depth;
  if (!next) next = (cu_attr_t)real_dlsym()(RTLD_NEXT, "cuDeviceGetAttribute");
  if (!next) {
    void *h = dlopen("libcuda.so.1", RTLD_NOW | RTLD_NOLOAD);
    if (h) next = (cu_attr_t)real_dlsym()(h, "cuDeviceGetAttribute");
  }
  if (!next || next == cuDeviceGetAttribute || depth) return 999;
  depth++;
  int r = next(value, attr, dev), major;
  if (!r && attr == 16 && value && *value > 0 && !strncmp(program_invocation_short_name, "FahCore_", 8)) {
    if (blocks < 0) blocks = checkpoint_blocks();
    int per = next(&major, 75, dev) ? 6 : major == 6 ? 4 : 6;
    long sms = blocks / per;
    if (blocks > 0 && blocks % per == 0 && sms > *value && sms <= 4L * *value) {
      static int told;
      if (!told++)
        fprintf(stderr, "fakepci: checkpoint was written for %ld thread blocks; reporting %ld SMs instead of %d\n",
                blocks, sms, *value);
      *value = (int)sms;
    }
  }
  depth--;
  return r;
}

int open64(const char *path, int flags, ...) {
  char b[4096]; REAL(open64);
  mode_t mode = 0;
  if ((flags & O_CREAT) || (flags & O_TMPFILE) == O_TMPFILE) { va_list ap; va_start(ap, flags); mode = va_arg(ap, mode_t); va_end(ap); }
  return real(redirect(path, b, sizeof(b)), flags, mode);
}
