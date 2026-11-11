"""Local Alertmanager webhook receiver. No external messages or credentials.

Demo only: JSONL in DATA_DIR survives restarts when backed by a volume.
Keep this unauthenticated UI internal; it is not a production notification channel.
"""
import html
import json
import os
import threading
from datetime import datetime, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

DATA = Path(os.environ.get('DATA_DIR', '/data'))
LOCK = threading.Lock()


class Handler(BaseHTTPRequestHandler):
    def reply(self, code, body, kind='application/json'):
        self.send_response(code)
        self.send_header('Content-Type', kind + '; charset=utf-8')
        self.end_headers()
        self.wfile.write(body.encode())

    def do_POST(self):
        if self.path != '/webhook':
            return self.reply(404, '{}')
        try:
            length = int(self.headers.get('Content-Length', '0'))
            if not 0 < length <= 1024 * 1024:
                return self.reply(413, '{}')
            payload = json.loads(self.rfile.read(length))
            if (not isinstance(payload, dict)
                    or payload.get('status') not in ('firing', 'resolved')
                    or not isinstance(payload.get('alerts'), list)
                    or any(not isinstance(a, dict)
                           or not isinstance(a.get('labels'), dict)
                           or not isinstance(a.get('annotations', {}), dict)
                           for a in payload['alerts'])):
                return self.reply(400, '{}')
        except (ValueError, UnicodeError):
            return self.reply(400, '{}')

        event = {
            'received_at': datetime.now(timezone.utc).isoformat(),
            'payload': payload,
        }
        with LOCK:
            DATA.mkdir(parents=True, exist_ok=True)
            with (DATA / 'notifications.jsonl').open('a') as f:
                f.write(json.dumps(event, ensure_ascii=False) + '\n')
                f.flush()
                os.fsync(f.fileno())
        self.reply(200, '{"accepted":true}')

    def do_GET(self):
        if self.path == '/healthz':
            return self.reply(200, '{"status":"ok"}')
        if self.path not in ('/', '/events'):
            return self.reply(404, '{}')

        with LOCK:
            path = DATA / 'notifications.jsonl'
            events = [
                json.loads(line) for line in path.read_text().splitlines()
            ] if path.exists() else []

        if self.path == '/events':
            return self.reply(200, json.dumps(events, ensure_ascii=False))

        cards = []
        esc = lambda value: html.escape(str(value))
        for event in reversed(events[-50:]):
            payload = event['payload']
            alerts = ''.join(
                '<h3>' + esc(a['labels'].get('alertname', '')) + '</h3>'
                '<p>' + esc(a.get('annotations', {}).get('summary', '')) + '</p>'
                '<p>' + esc(a.get('annotations', {}).get('description', '')) + '</p>'
                '<pre>' + esc(json.dumps(
                    a['labels'], ensure_ascii=False, indent=2
                )) + '</pre>'
                for a in payload['alerts']
            )
            status = esc(payload.get('status', 'unknown'))
            cards.append(
                '<article><strong class="' + status + '">'
                + status.upper() + '</strong>'
                '<span> · ' + esc(event['received_at']) + '</span>'
                + alerts + '</article>'
            )

        page = '''<!doctype html><html lang="ru"><meta charset="utf-8">
<title>Repair Desk — уведомления Alertmanager</title>
<style>
body{font:16px system-ui;margin:40px auto;max-width:1100px;background:#f3f6fa;color:#17243b}
h1{font-size:30px}
article{background:white;border:1px solid #d6dfea;border-radius:10px;padding:22px;margin:18px 0}
.firing{color:#bd2424}
.resolved{color:#087b4b}
span{color:#58677a}
pre{background:#eef2f7;padding:12px}
p{line-height:1.5}
</style>
<h1>Repair Desk · уведомления Alertmanager</h1>
<p>Полученные HTTP webhook · FIRING — проблема · RESOLVED — восстановление · время UTC</p>'''
        self.reply(200, page + ''.join(cards) + '</html>', 'text/html')


if __name__ == '__main__':
    ThreadingHTTPServer(('0.0.0.0', 8080), Handler).serve_forever()