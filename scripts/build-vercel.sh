#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
mkdir -p vercel-public/static
cp -R static/. vercel-public/static/
find vercel-public/static -name '*.test.cjs' -delete
cp static/sw.js vercel-public/sw.js

# The public landing page has no Go template directives; keep private pages on Render.
cp templates/landing.html vercel-public/index.html

# Account entry shells contain no private data or Go template directives.
for page in login signup forgot-password reset-password; do
    mkdir -p "vercel-public/page/$page"
    cp "templates/$page.html" "vercel-public/page/$page/index.html"
done
