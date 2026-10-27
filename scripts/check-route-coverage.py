#!/usr/bin/env python3
"""Lists server routes alongside the clients that mention them.

Usage: check-route-coverage.py [--group admin|me] [--only-missing] [--exit-nonzero]

An endpoint that exists only on the server is a half-finished feature: the
capability is real, documented and tested, and no operator or owner can reach it.
Nothing about that shape looks broken - the route answers correctly when curled,
and the missing caller is invisible from either side.

This is a *reporting* tool rather than a gate, and the distinction is deliberate.
Whether a route is genuinely reachable is a question about code paths, and no
string search can answer it: the word "storage" appears in a client for reasons
that have nothing to do with `/recordings:storage`, so an automatic verdict
either cries wolf or stays quiet on a real gap. What this does instead is put the
route table and its candidate callers side by side, which is the comparison a
person has to make anyway - only without re-deriving the route list each time.

`--exit-nonzero` turns it into a gate over the *base* signal only, which is
coarse but safe: it catches a route whose whole prefix no client mentions, and
stays silent about the per-action cases this cannot judge.
"""
import argparse
import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MAIN_GO = os.path.join(ROOT, "SignalingServer", "cmd", "signaling", "main.go")

# The receiver each group is registered under in main.go, and which client trees
# may legitimately call it. The receiver name is not always the group name:
# admin routes hang off `adminGuarded`, a router group rather than a handler.
GROUPS = {
    "admin": {
        "receiver": "adminGuarded",
        "trees": ["SignalingServer/web/src"],
    },
    "me": {
        "receiver": "me",
        "trees": ["WebClient/js", "WebClient/src", "QuickDesk/src", "QuickDesk/qml"],
    },
}

READABLE_SUFFIXES = (".js", ".vue", ".cpp", ".h", ".qml", ".ts")
SKIP_DIRS = {"node_modules", "dist", "target", ".git"}


def load_routes(group):
    with open(MAIN_GO, encoding="utf-8") as handle:
        source = handle.read()
    # me.GET("/x") / adminGuarded.POST("/x") - the group name is the receiver.
    pattern = re.compile(r'\b%s\.(?:GET|POST|PUT|PATCH|DELETE)\("([^"]+)"' % re.escape(group))
    return sorted(set(pattern.findall(source)))


def route_base(path):
    """The part of a route a client would hard-code.

    `/recordings/:id/artifact` -> `/recordings`, because a client writes
    `${RECORDING_BASE}/${id}/artifact` and never spells the literal path.
    `:search` style suffixes are dropped for the same reason.
    """
    base = path.split(":", 1)[0]
    base = re.sub(r"/[:\*][^/]*", "", base)
    return base.rstrip("/") or "/"


def read_clients(trees):
    blob = []
    for tree in trees:
        absolute = os.path.join(ROOT, tree)
        for directory, subdirs, files in os.walk(absolute):
            subdirs[:] = [d for d in subdirs if d not in SKIP_DIRS]
            for name in files:
                if not name.endswith(READABLE_SUFFIXES):
                    continue
                path = os.path.join(directory, name)
                try:
                    with open(path, encoding="utf-8", errors="ignore") as handle:
                        blob.append(handle.read())
                except OSError:
                    continue
    return "\n".join(blob)


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--group", choices=sorted(GROUPS), default="admin")
    parser.add_argument("--only-missing", action="store_true", help="list only the gaps")
    parser.add_argument("--exit-nonzero", action="store_true",
                        help="fail when a route's base is mentioned nowhere")
    args = parser.parse_args()

    spec = GROUPS[args.group]
    routes = load_routes(spec["receiver"])
    if not routes:
        print("no routes found for group %r; is the receiver still called %r?" % (
            args.group, spec["receiver"]))
        return 2

    haystack = read_clients(spec["trees"])

    covered, missing = [], []
    for route in routes:
        base = route_base(route)
        (covered if base in haystack else missing).append((route, base))

    print("%s: %d routes, %d covered, %d with no client" % (
        args.group, len(routes), len(covered), len(missing)))

    if not args.only_missing:
        print("")
        print("routes and the base a client would use to reach them:")
        for route, base in covered:
            print("  ok      %s  (base %s)" % (route, base))

    if missing:
        print("")
        print("bases no client mentions:")
        for route, base in missing:
            print("  MISSING %s  (no client mentions %r)" % (route, base))

    if missing and args.exit_nonzero:
        print("")
        print("A base nothing mentions is a route no client can reach. Either wire")
        print("it up, or record in the checklist why it is deliberately server-only.")
        return 1

    print("")
    print("This is a report, not a verdict: a base appearing somewhere does not")
    print("prove the action is reachable, and its absence from a client tree is")
    print("the only thing this can assert on its own.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
