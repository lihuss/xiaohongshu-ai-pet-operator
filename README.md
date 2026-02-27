# xiaohongshu-ai-pet-operator

Security control layer in front of `xiaohongshu-mcp` so the AI pet only executes commands from one bound owner account.

## Security model

- Owner identity is bound to immutable `owner_user_id`, not nickname.
- Nickname changes do not affect ownership if `owner_user_id` stays the same.
- Owner-change commands are blocked by default:
  - `change_owner`
  - `transfer_owner`
  - `reset_owner`
  - `bind_owner`
- Every command requires HMAC signature with `OWNER_SHARED_SECRET`.
- Every command requires `timestamp + nonce` to prevent replay attacks.
- Only allowlisted commands can be executed.

## Project layout

- `cmd/server`: entrypoint
- `internal/server`: HTTP API
- `internal/security`: signature and anti-replay
- `internal/owner`: owner lock and forbidden commands
- `internal/xhs`: allowlisted proxy calls to `xiaohongshu-mcp`

## Environment variables

- `OWNER_USER_ID`: required owner unique id (XHS userId)
- `OWNER_SHARED_SECRET`: required signing secret (32+ random chars recommended)
- `MCP_BASE_URL`: upstream MCP API base URL, default `http://127.0.0.1:18060`
- `LISTEN_ADDR`: this service address, default `:8081`

Use `.env.example` as a template.

## Run

```bash
go run ./cmd/server
```

## Sign request (PowerShell)

```powershell
.\scripts\sign-command.ps1 `
  -ActorUserId "your_owner_user_id" `
  -Command "check_login_status" `
  -Secret "your_OWNER_SHARED_SECRET" `
  -ArgsJson "{}"
```

Then call:

```powershell
$body = .\scripts\sign-command.ps1 -ActorUserId "your_owner_user_id" -Command "check_login_status" -Secret "your_OWNER_SHARED_SECRET"
Invoke-RestMethod -Uri "http://127.0.0.1:8081/v1/command" -Method Post -Body $body -ContentType "application/json"
```

## Allowlisted commands

- `check_login_status`
- `my_profile`
- `list_feeds`
- `search_feeds`
- `feed_detail`
- `user_profile`
- `publish_content`
- `publish_video`
- `post_comment`
- `reply_comment`

To add commands, edit `allowlist` in `internal/xhs/client.go`.

