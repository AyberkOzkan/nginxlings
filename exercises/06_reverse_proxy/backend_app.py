#!/usr/bin/env python3
from http.server import BaseHTTPRequestHandler, HTTPServer


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        body = b"backend-06-ok\n"
        self.send_response(200)
        self.send_header("Content-Type", "text/plain; charset=utf-8")
        self.send_header("X-Backend", "exercise-06")
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, format, *args):
        # Keep backend quiet in the terminal
        pass


def main():
    server = HTTPServer(("127.0.0.1", 8080), Handler)
    server.serve_forever()


if __name__ == "__main__":
    main()

