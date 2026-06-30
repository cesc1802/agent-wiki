# PRD — Agent Wiki (nvtwiki)

| | |
|---|---|
| **Tên sản phẩm** | nvtwiki — Agent-maintained Project Knowledge Wiki |
| **Chủ sở hữu** | Nguyễn Văn Thược (Delivery Manager / Backend Leader) |
| **Phiên bản tài liệu** | v0.1 (draft) |
| **Ngày** | 2026-06-29 |
| **Trạng thái** | Đang chốt scope — chờ phê duyệt để vào Phase A |

---

## 1. Tóm tắt (Executive Summary)

nvtwiki là một CLI tool viết bằng Go, đóng vai trò **orchestrator** điều khiển Claude Code (qua `claude -p` headless mode) để xây dựng và bảo trì một knowledge base dạng wiki cho các dự án phần mềm.

Vấn đề cần giải: tài liệu phát triển và các quyết định trong quá trình làm dự án (plan, phase, design, ADR) nằm rải rác dưới dạng markdown trong repo, khó tra cứu, dễ lỗi thời, và mất ngữ cảnh khi onboard người mới hoặc maintain dự án lâu dài.

Giải pháp: một hệ thống 3 layer (raw sources bất biến → wiki do LLM sở hữu → schema điều khiển LLM), với CLI làm harness deterministic để **chuẩn bị context, ép đúng quy trình, và validate kết quả**, còn Claude Code lo phần đọc-hiểu-viết-tổng hợp.

**Nguyên tắc kiến trúc cốt lõi:** CLI deterministic chịu trách nhiệm phần cơ học và các gate kiểm soát; Claude Code chịu trách nhiệm phần ngữ nghĩa. Đây là pattern hard-gate FSM — CLI là gate, Claude là executor.

---

## 2. Mục tiêu (Goals)

### 2.1. Mục tiêu sản phẩm

1. **Ingest** — đưa một nguồn mới (plan/phase markdown) vào wiki: sinh summary page, cập nhật entity/concept pages liên quan, duy trì cross-reference, ghi log.
2. **Query** — tra cứu kiến thức dự án, nhận câu trả lời có citation; câu trả lời giá trị được "file-back" thành page mới để tích lũy.
3. **Lint** — health-check wiki định kỳ: phát hiện mâu thuẫn, claim lỗi thời, orphan page, concept thiếu page, link gãy.
4. **Onboarding & maintain** — đủ thông tin có cấu trúc để người mới tự tra cứu và đội ngũ bảo trì dự án dài hạn.

### 2.2. Mục tiêu kỹ thuật

1. Mỗi project là một wiki độc lập (index/log riêng) — tránh `index.md` phình vượt context window.
2. Chống decay bằng cơ chế `status` + `superseded_by` ở tầng convention, không phụ thuộc hạ tầng.
3. Không dùng RAG / vector DB — dùng `index.md` làm điểm vào điều hướng (đủ tốt ở quy mô ~100 nguồn, hàng trăm page).
4. Tự động hóa qua `claude -p` nhưng giữ kiểm soát con người ở tầng review diff (git).

### 2.3. Tiêu chí thành công (Success Criteria)

- Ingest một plan thật của Klyx sinh đủ page liên quan, validate + lint cơ học pass, git diff sạch và review được.
- Query một câu hỏi onboarding điển hình trả về câu trả lời đúng, có citation về nguồn raw.
- Lint phát hiện đúng orphan page và link gãy trên wiki thật.
- Không có đường nào để agent ghi/sửa vào `raw/` (kiểm chứng bằng test đối kháng).

---

## 3. Phi mục tiêu (Non-Goals)

Các mục dưới đây **chủ ý không** nằm trong sản phẩm, để tránh over-engineering:

1. **Không xây dựng RAG / embedding / vector database.** Đây là quyết định kiến trúc có chủ đích. `index.md` thay thế retrieval ở quy mô mục tiêu.
2. **Không tự động hóa hoàn toàn không giám sát.** Mọi thay đổi wiki phải qua git diff review trước khi commit. Không auto-commit, không auto-push.
3. **Không đồng bộ ngược lên Jira/Notion/Confluence.** Markdown trong git là source of truth duy nhất. Tránh bài toán đồng bộ hai chiều.
4. **CLI không tự sửa nội dung page.** CLI chỉ scaffold file rỗng/stub; nội dung là lãnh địa độc quyền của Claude.
5. **CLI không tự gọi LLM cho phần lint cơ học và validate.** Chỉ ba lệnh ingest/query/lint-semantic gọi `claude -p`.
6. **Không xử lý quyền truy cập đa người dùng / RBAC.** Công cụ cá nhân / team nhỏ, chạy local.

---

## 4. Phạm vi (Scope)

### 4.1. In-Scope (MVP → v1)

| Hạng mục | Mô tả |
|---|---|
| CLI scaffolding | `init`, `project add/list` |
| Validate frontmatter | Kiểm schema frontmatter mọi page (deterministic) |
| Lint cơ học | Orphan, link gãy, `superseded_by` sai/vòng lặp, status mâu thuẫn (deterministic) |
| Navigation | `status`, `log`, `raw` (liệt kê nguồn chưa ingest) |
| Ingest qua `claude -p` | Dựng prompt + gọi headless + gate validate/lint sau chạy |
| Query qua `claude -p` | Read-only + file-back tùy chọn |
| Lint ngữ nghĩa qua `claude -p` | Mâu thuẫn nội dung, stale claim, concept thiếu page |
| Permission harness | PreToolUse hook chặn write ngoài `wiki/`; permission profile per-command |
| Cost/budget tracking | Parse `total_cost_usd`, cộng dồn, `--max-budget` |
| CLAUDE.md (schema) | File điều khiển hành vi agent — đã soạn xong v1 |

### 4.2. Out-of-Scope (giai đoạn này)

| Hạng mục | Lý do hoãn/loại |
|---|---|
| Vector search / RAG | Không cần ở quy mô mục tiêu; mâu thuẫn triết lý index-based |
| Web UI / dashboard | CLI-first; chưa cần |
| Multi-user / RBAC | Công cụ cá nhân/team nhỏ |
| Đồng bộ Jira/Notion/Confluence | Git là source of truth duy nhất |
| Auto-commit / auto-push | Giữ human review qua git diff |
| Batch-ingest qua đêm (cron) | Cân nhắc sau v1, khi harness đã vững |
| Tách index theo category khi scale lớn | Chỉ cần khi vượt ~vài trăm page/project |
| Resume session đa lượt (`--resume`) | Mỗi thao tác là một session độc lập ở v1 |

---

## 5. Kiến trúc hệ thống

### 5.1. Mô hình 3 layer

```
knowledge-base/
├── CLAUDE.md                    # SCHEMA — điều khiển agent, dùng chung mọi project
├── nvtwiki                      # binary CLI (Go) — orchestrator
├── schema.yaml                  # nguồn sự thật DUY NHẤT cho frontmatter schema
├── nvtwiki.yaml                 # config: danh sách project, đường dẫn
├── hooks/
│   └── block-raw-write.sh       # PreToolUse hook — chặn write ngoài wiki/
└── projects/
    ├── klyx/
    │   ├── raw/                 # NGUỒN BẤT BIẾN — chỉ đọc
    │   └── wiki/                # LLM SỞ HỮU HOÀN TOÀN
    │       ├── index.md         # catalog — đọc đầu tiên khi query
    │       ├── log.md           # timeline append-only
    │       ├── overview.md
    │       ├── entities/
    │       ├── concepts/
    │       ├── sources/
    │       └── synthesis/
    └── dtmyx/
        ├── raw/
        └── wiki/...
```

### 5.2. Phân định trách nhiệm CLI ↔ Claude

| Thao tác | CLI (deterministic) | Claude (ngữ nghĩa) |
|---|---|---|
| Scaffold cấu trúc, thư mục | ✅ | |
| Validate cú pháp frontmatter | ✅ | |
| Duyệt graph link, tìm orphan/link gãy | ✅ | |
| So field `status`/`superseded_by` | ✅ | |
| Query log, thống kê, liệt kê raw | ✅ | |
| Đọc hiểu nội dung nguồn | | ✅ |
| Viết/cập nhật page | | ✅ |
| Phát hiện mâu thuẫn **nội dung** | | ✅ |
| Quyết định file-back | | ✅ |
| Tổng hợp câu trả lời | | ✅ |

**Invariant cứng:** CLI không bao giờ sửa nội dung page; Claude không bao giờ ghi vào `raw/`.

### 5.3. Luồng orchestration (ví dụ: ingest)

```
nvtwiki ingest klyx raw/klyx/phase-2-plan.md
  │
  ├─[1] CLI dựng prompt + chọn cwd = projects/klyx/
  ├─[2] CLI gọi claude -p với permission profile khóa chặt
  ├─[3] Claude tự đọc CLAUDE.md (ở trong cwd) → chạy workflow Ingest 9 bước → tự ghi page
  ├─[4] CLI parse JSON output {result, session_id, total_cost_usd}
  ├─[5] CLI chạy validate + lint cơ học (GATE) → fail thì báo bước thiếu
  └─[6] CLI in report + cost; bạn git diff review trước khi commit
```

---

## 6. Đặc tả convention dữ liệu

### 6.1. Frontmatter (nguồn sự thật: `schema.yaml`)

Mọi page trong `wiki/` (trừ `index.md`, `log.md`) bắt buộc có:

```yaml
---
title: Tên page
type: entity | concept | source | synthesis | overview
project: klyx | dtmyx | shared
status: active | superseded | archived
superseded_by: null              # đường dẫn page thay thế nếu có
sources:                         # nguồn trong raw/ làm cơ sở
  - raw/klyx/phase-2-plan.md
created: YYYY-MM-DD
updated: YYYY-MM-DD
---
```

### 6.2. Cross-reference

Wikilink kèm đường dẫn thực (vì Claude Code thao tác trên path, không có resolver `[[ ]]`):

```markdown
Quyết định dựa trên [[hard-gate-fsm]](../concepts/hard-gate-fsm.md).
```

### 6.3. index.md — catalog (content-oriented)

Catalog mọi page theo category (Overview/Entities/Concepts/Sources/Synthesis), mỗi dòng: wikilink + path + tóm tắt một dòng + metadata. Cập nhật mỗi lần ingest và file-back.

### 6.4. log.md — timeline (append-only)

Prefix chuẩn cho phép grep: `## [YYYY-MM-DD] <op> | <title>` với `op` ∈ {ingest, query, lint}.

```
grep "^## \[" wiki/log.md | tail -5
```

### 6.5. Ngôn ngữ

Nội dung tiếng Việt; thuật ngữ kỹ thuật giữ tiếng Anh. Không bịa khi nguồn không đề cập.

---

## 7. Đặc tả chức năng CLI

### 7.1. Nhóm deterministic (không gọi LLM)

| Lệnh | Chức năng |
|---|---|
| `init` | Scaffold knowledge-base: CLAUDE.md, schema.yaml, config, hook |
| `project add <name>` | Tạo project với đầy đủ `raw/` + `wiki/{...}` |
| `project list` | Liệt kê project + số nguồn/page |
| `validate <project>` | Kiểm frontmatter mọi page theo `schema.yaml` |
| `lint <project>` | Lint cơ học: orphan, link gãy, superseded sai |
| `status <project>` | Thống kê page theo type/status, nguồn đã/chưa ingest |
| `log <project> [-n N] [--op ingest\|query\|lint]` | Tail/lọc timeline |
| `raw <project>` | Liệt kê file raw chưa có source page (nhắc nợ ingest) |

### 7.2. Nhóm orchestration (gọi `claude -p`)

| Lệnh | Permission profile | Ghi chú |
|---|---|---|
| `ingest <project> <raw-file>` | allowedTools `Read,Write,Edit,Glob,Grep`; mode `acceptEdits`; max-turns ~30 | Gate validate/lint sau chạy |
| `query <project> "<câu hỏi>" [--save]` | mặc định read-only (`Read,Glob,Grep`); `--save` bật write cho file-back | File-back là bước hai, opt-in |
| `lint <project> --semantic` | read-only | Lint ngữ nghĩa; không ghi, chỉ report |

### 7.3. Permission profile per-command

| Lệnh | allowedTools | permission-mode | max-turns | Phạm vi ghi |
|---|---|---|---|---|
| `ingest` | Read,Write,Edit,Glob,Grep | acceptEdits | ~30 | `wiki/` (10–15 page) |
| `query` | Read,Glob,Grep | dontAsk | ~15 | không (trừ `--save`) |
| `lint --semantic` | Read,Glob,Grep | dontAsk | ~20 | không |

---

## 8. An toàn & kiểm soát (Safety & Guardrails)

### 8.1. Bảo vệ source of truth

- **Deny rule cứng** chặn mọi write vào `raw/**`. Deny rule là hard guarantee (boundary trong prompt có thể mất khi context compaction).
- **PreToolUse hook** (`hooks/block-raw-write.sh`) chặn ở tầng code, không phụ thuộc model judgment. Cài sẵn khi `init`.
- Không cấp tool `Bash` cho ingest/query/lint (chặn `rm`, `git push`, code execution).

### 8.2. Kiểm soát chi phí

- `--max-turns` chống agent loop ngốn quota bất ngờ (ingest đụng nhiều page).
- `--max-budget-usd` per-run; parse `total_cost_usd` từ JSON output, cộng dồn.
- Nếu repeated blocks → session tự abort (đặc tính `-p` mode), CLI báo lỗi.

### 8.3. Human-in-the-loop (đã dịch chuyển)

Đánh đổi của autonomous mode: kiểm soát con người chuyển từ **"duyệt trước từng quyết định"** sang **"review diff sau khi chạy"**.

- Mọi thay đổi nằm trong git; bạn `git diff` 10–15 page trước khi commit.
- **Git là human-in-the-loop mới.** Không auto-commit.
- Gate kép sau chạy: CLI tự `validate` + `lint`; fail → ingest coi như chưa hoàn tất.

---

## 9. Rủi ro & giảm thiểu (Risks)

| # | Rủi ro | Mức độ | Giảm thiểu |
|---|---|---|---|
| R1 | Drift giữa CLAUDE.md, validator (CLI), prompt template — ba nơi cùng mã hóa convention | Cao | `schema.yaml` là nguồn sự thật duy nhất; CLAUDE.md + CLI cùng tham chiếu. Chốt ở Phase A |
| R2 | Agent ghi nhầm vào `raw/` | Cao | Deny rule + PreToolUse hook; test đối kháng ở Phase B |
| R3 | Parser markdown sai (frontmatter/wikilink) → lint ra kết quả rác, mất niềm tin tool | Cao | Test parser kỹ trên dữ liệu thật trước khi qua Phase orchestration |
| R4 | Agent bỏ bước trong workflow ingest (sót 1 trong 10–15 page) | Trung bình | Gate validate/lint sau chạy; checklist 9 bước cứng trong CLAUDE.md |
| R5 | Quota/cost vượt kiểm soát do agent loop | Trung bình | `--max-turns` + `--max-budget-usd` từ Phase B |
| R6 | `index.md` phình vượt context khi scale lớn | Thấp (chưa tới ngưỡng) | Theo dõi; tách index theo category khi vượt ~vài trăm page |
| R7 | File-back bị Claude bỏ qua → explorations không tích lũy | Trung bình | Đánh dấu QUAN TRỌNG trong CLAUDE.md; theo dõi vài lần dùng đầu |

---

## 10. Kế hoạch triển khai (Roadmap)

> Thứ tự cố ý: deterministic trước, query (read-only, rủi ro thấp) trước ingest (write, rủi ro cao).

### Phase A — Deterministic core

- Lệnh: `init`, `project add/list`, `validate`, `lint` cơ học, `status/log/raw`.
- Chốt `schema.yaml` làm nguồn sự thật chung.
- **Xong khi:** dựng được knowledge-base + 2 project; validate/lint chạy đúng trên wiki thật.

### Phase B — Permission harness

- Viết PreToolUse hook chặn `raw/**`; test đối kháng (cố tình bảo claude sửa raw → bị chặn).
- Wrapper gọi `claude -p`, parse JSON output, áp `--max-turns` + `--max-budget`.
- **Xong khi:** gọi claude -p an toàn, không cách nào ghi vào raw.

### Phase C — Query (read-only)

- `query` đầy đủ: dựng context từ index → claude -p read-only → câu trả lời có citation + cost.
- File-back opt-in (`--save`) là bước hai.
- **Xong khi:** hỏi câu thật, nhận trả lời đúng có citation, cost hợp lý.

### Phase D — Ingest (write, rủi ro cao nhất)

- `ingest` write mode + gate validate/lint sau chạy + report diff.
- **Xong khi:** ingest 1 plan thật của Klyx, đủ page, gate pass, git diff sạch.

### Phase E — Lint semantic + budget + (tùy chọn) cron

- `lint --semantic`, cost tracking hoàn chỉnh, cân nhắc batch-ingest qua đêm.

---

## 11. Câu hỏi mở (Open Questions)

1. Harness in prompt ra stdout hay ghi `.prompt.md`? (Quyết định ở Phase B theo cách dùng thực tế.)
2. CLAUDE.md đặt ở root dùng chung, hay mỗi project một bản? (Ảnh hưởng cách Claude nạp context theo `cwd`.)
3. Ngưỡng cụ thể để tách `index.md` theo category — định lượng khi nào? (Theo dõi ở Phase E.)
4. Authentication cho `claude -p`: subscription seat (OAuth token) hay API key? (Ảnh hưởng cost model.)

---

## 12. Phụ lục — Quyết định kiến trúc đã chốt

| Quyết định | Lựa chọn | Lý do |
|---|---|---|
| Retrieval | index-based, KHÔNG RAG | Đủ tốt ở quy mô mục tiêu; vận hành nhẹ |
| Source of truth | markdown trong git | Versioning/provenance miễn phí; tránh đồng bộ hai chiều |
| Multi-project | mỗi project một wiki độc lập | Tránh index phình; context isolation tự nhiên qua `cwd` |
| Agent runtime | Claude Code qua `claude -p` headless | Tự nạp CLAUDE.md; có sẵn file tools; pattern đã quen |
| Vai trò CLI | orchestrator deterministic | Ép quy trình (gate), không gọi LLM cho phần cơ học |
| Cross-ref | wikilink + path thực | Vừa đọc đẹp vừa resolve được |
| Chống decay | `status` + `superseded_by` + lint | Convention-level, không phụ thuộc hạ tầng |
| Human control | review git diff sau chạy | Đánh đổi của autonomous mode |
