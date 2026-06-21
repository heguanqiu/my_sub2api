---
name: gpt-image
description: Generate or edit images with jlaudeapi.com gpt-image-2. Use for text-to-image, image edits, image transforms, or explicit gpt-image requests.
---

# GPT Image

Use this skill to generate or edit images through the `gpt-image-2` model on the jlaudeapi.com API. Keep API keys out of chat and command output.

## Compatibility

This skill follows the portable Agent Skills layout used by Codex, OpenClaw, Hermes, and compatible tools:

```text
gpt-image/
  SKILL.md
  scripts/
    gpt_image.py
```

Install the whole `gpt-image` directory into the skill root used by the target tool. Common locations:

- Codex: `~/.agents/skills/gpt-image` or a repository `.agents/skills/gpt-image`
- OpenClaw: `~/.openclaw/workspace/skills/gpt-image` or another configured OpenClaw skills root
- Hermes: `~/.hermes/skills/gpt-image`, or add a shared root such as `~/.agents/skills` to Hermes `skills.external_dirs`

## Requirements

- Python 3.9 or newer
- Network access to the API base URL
- `JLAUDE_API_KEY` set in the environment

Optional environment variables:

- `JLAUDE_BASE_URL`: API base URL. Default: `https://img-api.jlaudeapi.com/`
- `JLAUDE_IMAGE_DIR`: default output directory. Default: `./images`
- `JLAUDE_TIMEOUT_SECONDS`: request and download timeout. Default: `600`
- `JLAUDE_RESPONSE_FORMAT`: `url` or `b64_json`. Default: `url`

Never print the full API key. If the key is missing, tell the user to set `JLAUDE_API_KEY` and stop.

## Configure API Key

Set `JLAUDE_API_KEY` in the shell or app environment before using the skill.

Temporary setup for the current terminal:

```bash
export JLAUDE_API_KEY="sk-..."
```

```powershell
$env:JLAUDE_API_KEY = "sk-..."
```

Permanent setup on macOS/Linux:

```bash
echo 'export JLAUDE_API_KEY="sk-..."' >> ~/.zshrc
source ~/.zshrc
```

Permanent setup on Windows PowerShell:

```powershell
[Environment]::SetEnvironmentVariable("JLAUDE_API_KEY", "sk-...", "User")
```

Restart the terminal, Codex, OpenClaw, Hermes, or any host app after setting a permanent environment variable.

Optional overrides:

```bash
export JLAUDE_BASE_URL="https://img-api.jlaudeapi.com/"
export JLAUDE_IMAGE_DIR="./images"
```

```powershell
[Environment]::SetEnvironmentVariable("JLAUDE_BASE_URL", "https://img-api.jlaudeapi.com/", "User")
[Environment]::SetEnvironmentVariable("JLAUDE_IMAGE_DIR", ".\images", "User")
```

Quick check:

```bash
python scripts/gpt_image.py "test image of a red cube"
```

If the helper says `JLAUDE_API_KEY is not set`, the host app did not receive the environment variable. Restart the app or configure the key in that app's environment settings.

## User Options

Supported sizes:

- `1024x1024` (default)
- `1024x1792`
- `1792x1024`

Supported count:

- `--n <int>` defaults to `1`

Supported response handling:

- `--timeout <seconds>` defaults to `600`
- `--response-format url|b64_json` defaults to `url`

Supported modes:

- Generate from text: prompt only
- Edit an image: `edit <image-path> <prompt>`

Invocation varies by host. Use `$gpt-image` in Codex, `/gpt-image` in Hermes or OpenClaw, or ask naturally for image generation/editing.

## Procedure

1. Determine the mode from the user request.
   - If the request starts with `edit <file>`, use edit mode and treat the remaining text as the prompt.
   - Otherwise, use generate mode and treat the request as the prompt.
2. Extract options such as `--size`, `--n`, `--out-dir`, `--timeout`, and `--response-format`.
3. Resolve the helper script path relative to this `SKILL.md`: `scripts/gpt_image.py`. If the host supports `{baseDir}`, use `{baseDir}/scripts/gpt_image.py`; otherwise compute the path from the skill directory.
4. Note the command start time and the intended output directory before running the helper.
5. Run the helper with Python. Do not create the output directory separately in shell; the helper creates it.
6. Keep chat output minimal while the helper runs. Do not narrate routine progress.
7. Report only the generated image count and absolute saved paths from `Saved:` lines or newly created files in the output directory. In Codex desktop, include a Markdown image preview of the saved local file when the user asked to output the image.
8. Do not perform an extra visual confirmation or quality check after files are saved unless the user explicitly asks for it.

## Commands

Generate one image:

```bash
python scripts/gpt_image.py "a neon cityscape at night"
```

Generate multiple images:

```bash
python scripts/gpt_image.py "a dragon made of glass" --size 1792x1024 --n 2
```

Edit an existing image:

```bash
python scripts/gpt_image.py edit ./photo.png "make the sky pink" --size 1024x1024
```

Choose an output directory:

```bash
python scripts/gpt_image.py "minimal app icon, blue glass" --out-dir ./images
```

Use a longer timeout or base64 output if needed:

```bash
python scripts/gpt_image.py edit ./photo.png "make the sky pink" --size 1024x1024 --timeout 900 --response-format b64_json
```

On Windows, use `py` or `python` according to what is available:

```powershell
py scripts/gpt_image.py "a clean product mockup on a desk"
```

## Output Handling

The helper prints one `Saved:` line per local file. Treat those paths as the final artifacts.

If the helper output is interrupted, the HTTP connection closes, or the command exits with an error after a long-running request, treat the result as indeterminate instead of failed. Before retrying or reporting failure, check the intended output directory for files created after the command start time. If files exist, treat the generation as successful and report those paths.

If the API returns URLs instead of base64 image data, the helper attempts to download each URL into the output directory. If downloading fails, it prints the URL so the user can still access it.

## Failure Handling

- Missing API key: explain that `JLAUDE_API_KEY` must be set.
- Missing edit image: ask for a valid local image path.
- API error: show the API error message, not the full raw response unless needed for debugging.
- Timeout or remote disconnect: first check the output directory for newly saved files. Only call it a failure if no new file or URL is available.
- Empty response: report that no image data was returned.
