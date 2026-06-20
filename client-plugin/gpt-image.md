Generate or edit images using gpt-image-2 via the jlaudeapi.com API.
Supports text-to-image and image-to-image (edit). Works in any project.

---

## Install

Copy this file to `.claude/skills/gpt-image.md` in your project (or `~/.claude/skills/` for global use).

## One-time config

Add to your shell profile or `.env`:
```
export JLAUDE_API_KEY=sk-...
```
Optionally override the base URL:
```
export JLAUDE_BASE_URL=https://jlaudeapi.com   # default
```

---

## How to invoke

| Intent | Command |
|---|---|
| Text → image | `/gpt-image a cat sitting on the moon` |
| Image → image | `/gpt-image edit ./photo.png make the sky pink` |
| With options | `/gpt-image a dragon --size 1024x1792 --n 2` |

Supported `--size` values: `1024x1024` (default), `1024x1792`, `1792x1024`

---

## Instructions (follow these exactly when invoked)

### Step 1 — Read args

Parse the user's input after `/gpt-image`:
- If it starts with `edit <file> `, mode = **edit**, extract file path and the rest as prompt.
- Otherwise mode = **generate**, full text is the prompt.
- Extract optional flags: `--size <value>` (default `1024x1024`), `--n <int>` (default `1`).

### Step 2 — Check config

Run:
```bash
echo "${JLAUDE_API_KEY:0:4}..."
```
If empty, tell the user to set `JLAUDE_API_KEY` and stop.

Set:
```bash
BASE_URL="${JLAUDE_BASE_URL:-https://jlaudeapi.com}"
OUT_DIR="${JLAUDE_IMAGE_DIR:-./images}"
mkdir -p "$OUT_DIR"
```

### Step 3 — Call the API

**Mode: generate** (text → image)

```bash
RESPONSE=$(curl -s --max-time 300 \
  -H "Authorization: Bearer $JLAUDE_API_KEY" \
  -H "Content-Type: application/json" \
  -d "{\"model\":\"gpt-image-2\",\"prompt\":\"$PROMPT\",\"size\":\"$SIZE\",\"n\":$N,\"response_format\":\"b64_json\"}" \
  "$BASE_URL/v1/images/generations")
```

**Mode: edit** (image → image)

```bash
RESPONSE=$(curl -s --max-time 300 \
  -H "Authorization: Bearer $JLAUDE_API_KEY" \
  -F "model=gpt-image-2" \
  -F "prompt=$PROMPT" \
  -F "size=$SIZE" \
  -F "image=@$IMAGE_PATH" \
  "$BASE_URL/v1/images/edits")
```

> gpt-image-2 can take 1–3 minutes. The 300s timeout is intentional.

### Step 4 — Save images

Parse the response with Python and save each image:

```bash
python3 - "$OUT_DIR" <<'EOF'
import sys, json, base64, os, time

out_dir = sys.argv[1]
data = json.load(sys.stdin)

if "error" in data:
    print(f"API error: {data['error'].get('message', data['error'])}")
    sys.exit(1)

ts = int(time.time())
saved = []
for i, item in enumerate(data.get("data", [])):
    path = os.path.join(out_dir, f"gpt-image-{ts}-{i+1}.png")
    if "b64_json" in item:
        with open(path, "wb") as f:
            f.write(base64.b64decode(item["b64_json"]))
        saved.append(path)
    elif "url" in item:
        saved.append(item["url"])  # URL-mode: print URL, user downloads manually

for p in saved:
    print(p)
EOF
echo "$RESPONSE" | python3 ...
```

Concretely, pipe `$RESPONSE` into the script, capture stdout as the list of saved paths.

### Step 5 — Report

Tell the user:
- How many images were generated
- The full path(s) of the saved file(s)
- If it's a local file, offer to read it back as a preview (use the Read tool on the image)

If the response contains an error, show the API error message clearly.

---

## Example session

```
User: /gpt-image a neon cityscape at night --size 1792x1024

AI:  Calling gpt-image-2... (this may take 1-2 minutes)
     ✓ Saved: ./images/gpt-image-1750000000-1.png
```

```
User: /gpt-image edit ./photo.png turn it into an oil painting

AI:  Calling gpt-image-2 (edit mode)...
     ✓ Saved: ./images/gpt-image-1750000001-1.png
```
