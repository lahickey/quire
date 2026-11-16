// Timetable.cs - departures for a stop, with live delays merged in.
#nullable enable
using System;
using System.Collections.Generic;
using System.Globalization;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;

namespace Transit.Timetable;

public enum Mode { Bus, Tram, Rail }

public readonly record struct StopId(string Value)
{
    public override string ToString() => Value;
}

public sealed record Departure(StopId Stop, string Line, Mode Mode, TimeOnly Scheduled)
{
    public TimeSpan Delay { get; init; } = TimeSpan.Zero;
    public TimeOnly Expected => Scheduled.Add(Delay);
    public bool IsLate => Delay > TimeSpan.FromMinutes(2);
}

public interface IDelayFeed
{
    IAsyncEnumerable<(string Line, int Seconds)> ReadAsync(CancellationToken ct = default);
}

/// <summary>Holds the day's departures and answers questions about them.</summary>
public class Timetable
{
    private const int MaxResults = 20;
    private static readonly CultureInfo Culture = CultureInfo.InvariantCulture;

    private readonly Dictionary<StopId, List<Departure>> _byStop = new();
    private readonly object _gate = new();

    public event EventHandler<Departure>? Delayed;

    public int Count
    {
        get { lock (_gate) { return _byStop.Values.Sum(l => l.Count); } }
    }

    public void Add(Departure departure)
    {
        lock (_gate)
        {
            if (!_byStop.TryGetValue(departure.Stop, out var list))
            {
                _byStop[departure.Stop] = list = new List<Departure>();
            }
            list.Add(departure);
        }
    }

    /* The next departures from a stop, soonest first. */
    public IReadOnlyList<Departure> Next(StopId stop, TimeOnly from, Mode? only = null)
    {
        lock (_gate)
        {
            if (!_byStop.TryGetValue(stop, out var list)) return Array.Empty<Departure>();
            var query =
                from d in list
                where d.Expected >= from && (only is null || d.Mode == only)
                orderby d.Expected, d.Line
                select d;
            return query.Take(MaxResults).ToList();
        }
    }

    public async Task<int> ApplyAsync(IDelayFeed feed, CancellationToken ct = default)
    {
        var changed = 0;
        await foreach (var (line, seconds) in feed.ReadAsync(ct).ConfigureAwait(false))
        {
            if (seconds is < 0 or > 3600) continue;
            lock (_gate)
            {
                foreach (var list in _byStop.Values)
                {
                    for (var i = 0; i < list.Count; i++)
                    {
                        if (list[i].Line != line) continue;
                        list[i] = list[i] with { Delay = TimeSpan.FromSeconds(seconds) };
                        changed++;
                        if (list[i].IsLate) Delayed?.Invoke(this, list[i]);
                    }
                }
            }
        }
        return changed;
    }

    public static string Describe(Departure d) => d switch
    {
        { IsLate: true, Mode: Mode.Rail } => $"{d.Line} delayed {d.Delay.TotalMinutes:F0} min, platform may change",
        { IsLate: true } => $"{d.Line} delayed {d.Delay.TotalMinutes:F0} min",
        { Mode: Mode.Tram or Mode.Bus } => string.Format(Culture, "{0} at {1:HH\\:mm}", d.Line, d.Expected),
        _ => $@"{d.Line} ""on time"" at {d.Expected:HH\:mm}",
    };
}

internal static class Program
{
    private sealed class FixedFeed : IDelayFeed
    {
        public async IAsyncEnumerable<(string Line, int Seconds)> ReadAsync(
            [System.Runtime.CompilerServices.EnumeratorCancellation] CancellationToken ct = default)
        {
            yield return ("42", 300);
            await Task.Delay(10, ct);
            yield return ("T1", 45);
        }
    }

    private static async Task<int> Main(string[] args)
    {
        var stop = new StopId(args.Length > 0 ? args[0] : "central");
        var table = new Timetable();
        table.Delayed += (_, d) => Console.Error.WriteLine($"! {d.Line} is late");

        table.Add(new Departure(stop, "42", Mode.Bus, new TimeOnly(8, 15)));
        table.Add(new Departure(stop, "T1", Mode.Tram, new TimeOnly(8, 20)));
        table.Add(new Departure(stop, "IC 7", Mode.Rail, new TimeOnly(9, 0)));

        using var cts = new CancellationTokenSource(TimeSpan.FromSeconds(1.5));
        try
        {
            var changed = await table.ApplyAsync(new FixedFeed(), cts.Token);
            Console.WriteLine($"{changed} of {table.Count} changed");
        }
        catch (OperationCanceledException) when (cts.IsCancellationRequested)
        {
            return 0x02;
        }

        foreach (var d in table.Next(stop, new TimeOnly(8, 0)))
        {
            Console.WriteLine(Timetable.Describe(d));
        }
        char tick = '✓';
        decimal fare = 2.40m;
        Console.WriteLine($"{tick} fare {fare:C} x {1_000L}");
        return 0;
    }
}
