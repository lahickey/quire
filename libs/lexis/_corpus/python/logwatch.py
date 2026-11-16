#!/usr/bin/env python3
"""Tail a set of log files and count what matches.

Usage:
    logwatch.py [--since HOURS] PATTERN FILE...
"""
from __future__ import annotations

import argparse
import asyncio
import re
import sys
from collections import Counter, defaultdict
from dataclasses import dataclass, field
from datetime import datetime, timedelta
from pathlib import Path
from typing import Iterator, Protocol

LEVELS = ("debug", "info", "warning", "error")
LINE = re.compile(r"^(?P<when>\S+ \S+) \[(?P<level>\w+)\] (?P<text>.*)$")
DEFAULT_WINDOW = timedelta(hours=24)
MAX_LINE = 0x4000


class Sink(Protocol):
    def write(self, text: str) -> int: ...


@dataclass(frozen=True, slots=True)
class Record:
    when: datetime
    level: str
    text: str
    source: Path

    @property
    def is_problem(self) -> bool:
        return self.level in {"warning", "error"}

    def __str__(self) -> str:
        return f"{self.when:%H:%M:%S} {self.level.upper():<7} {self.text!r}"


@dataclass
class Tally:
    by_level: Counter[str] = field(default_factory=Counter)
    by_source: defaultdict[Path, int] = field(default_factory=lambda: defaultdict(int))
    first: datetime | None = None

    def add(self, record: Record) -> None:
        self.by_level[record.level] += 1
        self.by_source[record.source] += 1
        if self.first is None or record.when < self.first:
            self.first = record.when

    def total(self) -> int:
        return sum(self.by_level.values())


def parse(path: Path, *, encoding: str = "utf-8") -> Iterator[Record]:
    # One record for each line that is a log line.
    with path.open(encoding=encoding, errors="replace") as handle:
        for number, line in enumerate(handle, start=1):
            if len(line) > MAX_LINE:
                print(f"{path}:{number}: line too long, skipped", file=sys.stderr)
                continue
            if not (m := LINE.match(line.rstrip("\n"))):
                continue
            try:
                when = datetime.strptime(m["when"], "%Y-%m-%d %H:%M:%S")
            except ValueError as err:
                raise SystemExit(f"{path}:{number}: bad time: {err}") from None
            level = m["level"].lower()
            if level not in LEVELS:
                level = "info"
            yield Record(when, level, m["text"], path)


async def scan(path: Path, pattern: re.Pattern[str], since: datetime) -> list[Record]:
    def work() -> list[Record]:
        return [r for r in parse(path) if r.when >= since and pattern.search(r.text)]

    return await asyncio.to_thread(work)


async def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("pattern")
    parser.add_argument("files", nargs="+", type=Path)
    parser.add_argument("--since", type=float, default=DEFAULT_WINDOW.total_seconds() / 3600)
    args = parser.parse_args(argv)

    pattern = re.compile(args.pattern, re.IGNORECASE)
    since = datetime.now() - timedelta(hours=args.since)
    results = await asyncio.gather(*(scan(p, pattern, since) for p in args.files), return_exceptions=True)

    tally = Tally()
    for path, result in zip(args.files, results):
        if isinstance(result, BaseException):
            print(f"{path}: {result}", file=sys.stderr)
            continue
        for record in sorted(result, key=lambda r: r.when):
            tally.add(record)
            if record.is_problem:
                print(record)

    width = max((len(str(p)) for p in tally.by_source), default=0)
    for source, count in sorted(tally.by_source.items(), key=lambda kv: -kv[1]):
        share = count / tally.total() * 100 if tally.total() else 0.0
        print(f"{str(source):<{width}}  {count:>6,}  {share:5.1f}%")
    print("levels:", ", ".join(f"{k}={v}" for k, v in tally.by_level.most_common()) or None)
    return 0 if tally.by_level["error"] == 0 else 1


if __name__ == "__main__":
    raise SystemExit(asyncio.run(main()))
