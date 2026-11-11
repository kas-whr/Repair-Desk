"""Exercise the real webhook HTTP endpoint and durable notification history."""
import importlib.util
import json
import tempfile
import threading
import unittest
import urllib.error
import urllib.request
from pathlib import Path

spec = importlib.util.spec_from_file_location(
    'inbox',
    Path(__file__).resolve().parents[1]
    / 'k8s/monitoring/alerts/inbox.py',
)
inbox = importlib.util.module_from_spec(spec)
spec.loader.exec_module(inbox)


class WebhookTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        inbox.DATA = Path(self.temp.name)
        self.server = inbox.ThreadingHTTPServer(
            ('127.0.0.1', 0), inbox.Handler
        )
        self.thread = threading.Thread(
            target=self.server.serve_forever
        )
        self.thread.start()
        self.url = (
            'http://127.0.0.1:' + str(self.server.server_port)
        )

    def tearDown(self):
        self.server.shutdown()
        self.thread.join()
        self.server.server_close()
        self.temp.cleanup()

    def post(self, data):
        return urllib.request.urlopen(
            urllib.request.Request(
                self.url + '/webhook',
                data=data,
                headers={'Content-Type': 'application/json'},
            )
        )

    def test_firing_and_resolved_are_persisted(self):
        for status in ('firing', 'resolved'):
            payload = {
                'status': status,
                'alerts': [{
                    'labels': {'alertname': 'Demo'},
                    'annotations': {
                        'summary': '<script>alert(1)</script>'
                    },
                }],
            }
            with self.post(
                json.dumps(payload).encode()
            ) as response:
                self.assertEqual(response.status, 200)

        with urllib.request.urlopen(
            self.url + '/events'
        ) as response:
            events = json.load(response)

        self.assertEqual(
            [e['payload']['status'] for e in events],
            ['firing', 'resolved'],
        )
        self.assertEqual(
            len(
                (inbox.DATA / 'notifications.jsonl')
                .read_text()
                .splitlines()
            ),
            2,
        )

        with urllib.request.urlopen(self.url) as response:
            page = response.read().decode()

        self.assertIn('&lt;script&gt;', page)
        self.assertNotIn('<script>', page)

    def test_invalid_json_does_not_create_history(self):
        with self.assertRaises(
            urllib.error.HTTPError
        ) as error:
            self.post(b'not json')

        self.assertEqual(error.exception.code, 400)
        self.assertFalse(
            (inbox.DATA / 'notifications.jsonl').exists()
        )

    def test_missing_alerts_is_rejected(self):
        with self.assertRaises(
            urllib.error.HTTPError
        ) as error:
            self.post(b'{}')

        self.assertEqual(error.exception.code, 400)

    def test_health_does_not_require_history(self):
        with urllib.request.urlopen(
            self.url + '/healthz'
        ) as response:
            self.assertEqual(
                json.load(response), {'status': 'ok'}
            )


if __name__ == '__main__':
    unittest.main()