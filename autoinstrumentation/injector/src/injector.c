#include <stdint.h>
#include <unistd.h>

#define ALIGN (sizeof(size_t))
#define UCHAR_MAX 255
#define ONES ((size_t)-1 / UCHAR_MAX)
#define HIGHS (ONES * (UCHAR_MAX / 2 + 1))
#define HASZERO(x) ((x) - ONES & ~(x) & HIGHS)

#define JAVA_TOOL_OPTIONS_ENV_VAR_NAME "JAVA_TOOL_OPTIONS"
#define JAVA_TOOL_OPTIONS_REQUIRE                                        \
  "-javaagent:"                                                          \
  "/otel-auto-instrumentation-injector/instrumentation/jvm/javaagent.jar"

extern char **__environ;

size_t __strlen(const char *s) {
  const char *a = s;
  const size_t *w;
  for (; (uintptr_t)s % ALIGN; s++)
    if (!*s)
      return s - a;
  for (w = (const void *)s; !HASZERO(*w); w++)
    ;
  for (s = (const void *)w; *s; s++)
    ;
  return s - a;
}

char *__strchrnul(const char *s, int c) {
  size_t *w, k;

  c = (unsigned char)c;
  if (!c)
    return (char *)s + __strlen(s);

  for (; (uintptr_t)s % ALIGN; s++)
    if (!*s || *(unsigned char *)s == c)
      return (char *)s;
  k = ONES * c;
  for (w = (void *)s; !HASZERO(*w) && !HASZERO(*w ^ k); w++)
    ;
  for (s = (void *)w; *s && *(unsigned char *)s != c; s++)
    ;
  return (char *)s;
}

char *__strcpy(char *restrict dest, const char *restrict src) {
  const unsigned char *s = src;
  unsigned char *d = dest;
  while ((*d++ = *s++))
    ;
  return dest;
}

char *__strcat(char *restrict dest, const char *restrict src) {
  __strcpy(dest + __strlen(dest), src);
  return dest;
}

int __strcmp(const char *l, const char *r) {
  for (; *l == *r && *l; l++, r++)
    ;
  return *(unsigned char *)l - *(unsigned char *)r;
}

int __strncmp(const char *_l, const char *_r, size_t n) {
  const unsigned char *l = (void *)_l, *r = (void *)_r;
  if (!n--)
    return 0;
  for (; *l && *r && n && *l == *r; l++, r++, n--)
    ;
  return *l - *r;
}

char *__getenv(const char *name) {
  size_t l = __strchrnul(name, '=') - name;
  if (l && !name[l] && __environ)
    for (char **e = __environ; *e; e++)
      if (!__strncmp(name, *e, l) && l[*e] == '=')
        return *e + l + 1;
  return 0;
}

/*
 * Buffers of statically-allocated memory that we can use to safely return to
 * the program manipulated values of env vars without dynamic allocations.
 */
char cachedModifiedRuntimeOptionsValue[1012];

char *getenv(const char *name) {
  char *origValue = __getenv(name);
  int l = __strlen(name);

  char *javaToolOptionsVarName = JAVA_TOOL_OPTIONS_ENV_VAR_NAME;
  if (__strcmp(name, javaToolOptionsVarName) == 0) {
    if (__strlen(cachedModifiedRuntimeOptionsValue) == 0) {
      // No runtime environment variable has been requested before,
      // calculate the modified value and cache it.

      // Prepend our --require as the first item to the JAVA_TOOL_OPTIONS
      // string.
      char *javaToolOptionsRequire = JAVA_TOOL_OPTIONS_REQUIRE;
      __strcat(cachedModifiedRuntimeOptionsValue, javaToolOptionsRequire);
    }

    return cachedModifiedRuntimeOptionsValue;
  }

  return origValue;
}
