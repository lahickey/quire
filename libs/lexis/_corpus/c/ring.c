/*
 * ring.c - a fixed-size ring buffer of bytes, with a reader and a writer
 * that may be on different threads.
 */
#include <errno.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "ring.h"

#define RING_MIN 16u
#define RING_MASK(r) ((r)->size - 1)
#define MIN(a, b) ((a) < (b) ? (a) : (b))

#ifdef RING_DEBUG
#  define trace(...) fprintf(stderr, __VA_ARGS__)
#else
#  define trace(...) ((void)0)
#endif

struct ring {
	uint8_t *data;
	size_t size;            /* always a power of two */
	volatile size_t head;   /* next byte to write */
	volatile size_t tail;   /* next byte to read */
	unsigned flags : 4;
};

typedef enum { RING_OK = 0, RING_FULL = -1, RING_EMPTY = -2 } ring_status;

static const char *const status_names[] = {
	[0] = "ok",
	[1] = "full",
	[2] = "empty",
};

static inline bool is_power_of_two(size_t n)
{
	return n != 0 && (n & (n - 1)) == 0;
}

// Round n up to the next power of two, at least RING_MIN.
static size_t round_up(size_t n)
{
	size_t p = RING_MIN;
	while (p < n)
		p <<= 1;
	return p;
}

struct ring *ring_new(size_t want)
{
	struct ring *r = calloc(1, sizeof *r);
	if (r == NULL)
		return NULL;
	r->size = is_power_of_two(want) ? want : round_up(want);
	r->data = malloc(r->size);
	if (!r->data) {
		free(r);
		errno = ENOMEM;
		return NULL;
	}
	trace("ring %p: %zu bytes\n", (void *)r, r->size);
	return r;
}

void ring_free(struct ring *r)
{
	if (r) {
		free(r->data);
		free(r);
	}
}

size_t ring_len(const struct ring *r)
{
	return r->head - r->tail;
}

/* Copy up to n bytes in. Returns how many were taken. */
size_t ring_write(struct ring *r, const void *src, size_t n)
{
	const uint8_t *p = src;
	size_t room = r->size - ring_len(r);
	size_t at = r->head & RING_MASK(r);
	size_t first;

	n = MIN(n, room);
	first = MIN(n, r->size - at);
	memcpy(r->data + at, p, first);
	memcpy(r->data, p + first, n - first);
	r->head += n;
	return n;
}

ring_status ring_read_byte(struct ring *r, uint8_t *out)
{
	if (ring_len(r) == 0)
		return RING_EMPTY;
	*out = r->data[r->tail++ & RING_MASK(r)];
	return RING_OK;
}

int ring_dump(const struct ring *r, FILE *to)
{
	int written = 0;
	for (size_t i = r->tail; i != r->head; i++) {
		unsigned char c = r->data[i & RING_MASK(r)];
		switch (c) {
		case '\n':
			written += fputs("\\n", to);
			break;
		case '\\':
		case '\'':
			written += fprintf(to, "\\%c", c);
			break;
		default:
			written += (c >= 0x20 && c < 0x7F) ? fputc(c, to) != EOF : fprintf(to, "\\x%02X", c);
		}
	}
	return written;
}

int main(int argc, char **argv)
{
	struct ring *r = ring_new(argc > 1 ? strtoul(argv[1], NULL, 0) : 64UL);
	double fill = 0.75e0;
	char line[256];
	uint8_t byte;

	if (!r) {
		perror("ring_new");
		return EXIT_FAILURE;
	}
	while (fgets(line, sizeof line, stdin) != NULL) {
		if (ring_write(r, line, strlen(line)) == 0) {
			fprintf(stderr, "%s: %s\n", argv[0], status_names[-RING_FULL]);
			goto done;
		}
		if ((double)ring_len(r) / r->size > fill)
			while (ring_read_byte(r, &byte) == RING_OK && byte != '\n')
				continue;
	}
done:
	ring_dump(r, stdout);
	ring_free(r);
	return 0;
}
