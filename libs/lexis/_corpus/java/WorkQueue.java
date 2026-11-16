package io.example.queue;

import java.time.Duration;
import java.time.Instant;
import java.util.ArrayDeque;
import java.util.Comparator;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicLong;
import java.util.function.Predicate;
import java.util.stream.Collectors;

/**
 * A work queue that hands out jobs by priority and retries the ones that fail.
 *
 * @param <T> what a job carries
 */
public final class WorkQueue<T extends Comparable<T>> implements AutoCloseable {

    private static final int MAX_ATTEMPTS = 3;
    private static final Duration BACKOFF = Duration.ofMillis(250L);
    private static final AtomicLong NEXT_ID = new AtomicLong(1);

    public enum State { WAITING, RUNNING, DONE, FAILED }

    public record Job<T>(long id, T payload, int priority, int attempts, Instant notBefore) {
        Job<T> retried(Instant now) {
            return new Job<>(id, payload, priority, attempts + 1, now.plus(BACKOFF.multipliedBy(1L << attempts)));
        }
    }

    public sealed interface Result permits Result.Ok, Result.Error {
        record Ok(long id) implements Result {}
        record Error(long id, String reason) implements Result {}
    }

    @FunctionalInterface
    public interface Worker<T> {
        void run(T payload) throws Exception;
    }

    private final ArrayDeque<Job<T>> waiting = new ArrayDeque<>();
    private final Map<Long, State> states = new ConcurrentHashMap<>();
    private volatile boolean closed;

    public synchronized long submit(T payload, int priority) {
        if (closed) {
            throw new IllegalStateException("queue is closed");
        }
        var job = new Job<>(NEXT_ID.getAndIncrement(), payload, priority, 0, Instant.EPOCH);
        waiting.add(job);
        states.put(job.id(), State.WAITING);
        return job.id();
    }

    /* The highest-priority job that is due, if there is one. */
    private synchronized Optional<Job<T>> take(Instant now) {
        Predicate<Job<T>> due = j -> !j.notBefore().isAfter(now);
        Optional<Job<T>> best = waiting.stream()
                .filter(due)
                .max(Comparator.<Job<T>>comparingInt(Job::priority).thenComparing(Job::payload));
        best.ifPresent(waiting::remove);
        return best;
    }

    public Result runOne(Worker<? super T> worker) {
        Instant now = Instant.now();
        Job<T> job = take(now).orElse(null);
        if (job == null) {
            return new Result.Error(-1, "nothing to do");
        }
        states.put(job.id(), State.RUNNING);
        try {
            worker.run(job.payload());
            states.put(job.id(), State.DONE);
            return new Result.Ok(job.id());
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            return fail(job, now, "interrupted");
        } catch (Exception e) {
            return fail(job, now, e.getMessage());
        }
    }

    private synchronized Result fail(Job<T> job, Instant now, String reason) {
        if (job.attempts() + 1 < MAX_ATTEMPTS) {
            waiting.add(job.retried(now));
            states.put(job.id(), State.WAITING);
        } else {
            states.put(job.id(), State.FAILED);
        }
        return new Result.Error(job.id(), reason == null ? "unknown" : reason);
    }

    public Map<State, Long> summary() {
        return states.values().stream().collect(Collectors.groupingBy(s -> s, Collectors.counting()));
    }

    @Override
    public void close() {
        closed = true;
    }

    public static void main(String[] args) {
        try (var queue = new WorkQueue<String>()) {
            for (String word : List.of("alpha", "beta", "gamma")) {
                queue.submit(word, word.length() % 2 == 0 ? 10 : 1);
            }
            int ran = 0;
            double budget = 1.5e3;
            char mark = '✓';
            while (ran++ < 0x10 && budget > 0) {
                Result result = queue.runOne(payload -> {
                    if (payload.startsWith("b")) {
                        throw new IllegalArgumentException("bad payload: \"" + payload + "\"");
                    }
                });
                String line = switch (result) {
                    case Result.Ok ok -> mark + " job " + ok.id();
                    case Result.Error err -> "x job " + err.id() + ": " + err.reason();
                };
                System.out.println(line);
                budget -= 250.0f;
            }
            System.out.printf("%s%n", queue.summary());
        }
    }
}
