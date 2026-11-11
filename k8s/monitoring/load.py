#!/usr/bin/env python3
"""Read-only demo traffic; stops after --seconds, bounded concurrency."""
import argparse
import concurrent.futures
import time
import urllib.error
import urllib.request
from collections import Counter


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--url', default='http://127.0.0.1:3000')
    parser.add_argument('--seconds', type=int, default=240)
    parser.add_argument('--workers', type=int, default=4)
    args = parser.parse_args()

    if args.seconds <= 0 or not 1 <= args.workers <= 20:
        parser.error('seconds must be positive; workers must be 1..20')

    end = time.monotonic() + args.seconds

    def worker():
        counts = Counter()
        while time.monotonic() < end:
            try:
                with urllib.request.urlopen(
                    args.url.rstrip('/') + '/api/v1/tickets?limit=20',
                    timeout=3,
                ) as response:
                    response.read()
                    counts[response.status] += 1
            except urllib.error.HTTPError as error:
                error.read()
                counts[error.code] += 1
            except (urllib.error.URLError, TimeoutError):
                counts['network_error'] += 1

            time.sleep(0.1)

        return counts

    total = Counter()
    with concurrent.futures.ThreadPoolExecutor(
        max_workers=args.workers
    ) as executor:
        for counts in executor.map(
            lambda _: worker(), range(args.workers)
        ):
            total.update(counts)

    print(dict(total))


if __name__ == '__main__':
    main()