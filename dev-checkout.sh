#!/usr/bin/env bash
# Switch repos to the correct branches and ensure all servers are running.
# Run from System_X_System/ or from System_X_System_Project/ (both work).
#
# Usage:
#   ./dev-checkout.sh <branch>                      # mesma branch para back e front
#   ./dev-checkout.sh <back-branch> <front-branch>  # branches diferentes
#
# Comportamento após trocar branches:
#   • Todos rodando    → air recompila back; Vite HMR recarrega front (nada a fazer)
#   • Só back offline  → inicia API + Game com hot reload neste terminal
#   • Só front offline → inicia Vite neste terminal
#   • Nenhum rodando   → inicia tudo (make dev) neste terminal
set -euo pipefail

BACK_BRANCH="${1:?Uso: ./dev-checkout.sh <back-branch> [front-branch]}"
FRONT_BRANCH="${2:-$BACK_BRANCH}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$SCRIPT_DIR"
FRONTEND_DIR="$SCRIPT_DIR/../System_X_System_React"
PROJECT_DIR="$SCRIPT_DIR/.."

GRN='\033[0;32m'; YLW='\033[1;33m'; RED='\033[0;31m'; CYN='\033[0;36m'
BLD='\033[1m'; NC='\033[0m'

ok()   { echo -e "  ${GRN}✓${NC} $*"; }
warn() { echo -e "  ${YLW}!${NC} $*"; }
sec()  { echo -e "\n${BLD}${CYN}▶ $*${NC}"; }

port_in_use() { lsof -ti:"$1" > /dev/null 2>&1; }

branch_exists() {
  local dir="$1" branch="$2"
  git -C "$dir" rev-parse --verify "refs/heads/$branch" &>/dev/null ||
  git -C "$dir" rev-parse --verify "refs/remotes/origin/$branch" &>/dev/null
}

find_worktree() {
  local dir="$1" branch="$2"
  local main_wt wt=""
  main_wt="$(git -C "$dir" worktree list --porcelain 2>/dev/null | grep '^worktree ' | head -1 | cut -d' ' -f2)"
  while IFS= read -r line; do
    case "$line" in
      "worktree "*) wt="${line#worktree }" ;;
      "branch refs/heads/$branch")
        [[ "$wt" != "$main_wt" ]] && echo "$wt"
        return 0 ;;
    esac
  done < <(git -C "$dir" worktree list --porcelain 2>/dev/null)
}

checkout_repo() {
  local name="$1" dir="$2" branch="$3" is_front="${4:-false}"

  if ! branch_exists "$dir" "$branch"; then
    warn "$name: branch '$branch' não encontrada — permanece na branch atual"
    return 0
  fi

  local worktree
  worktree="$(find_worktree "$dir" "$branch")"
  if [[ -n "$worktree" ]]; then
    warn "$name: '$branch' está em worktree — ${CYN}${worktree}${NC}"
    [[ "$is_front" == "true" ]] && \
      echo -e "         → Reinicie o Vite a partir desse path" || \
      echo -e "         → Reinicie o Go server a partir desse path"
    return 0
  fi

  local current
  current="$(git -C "$dir" branch --show-current 2>/dev/null || echo "")"

  if [[ "$current" != "$branch" ]]; then
    local err
    if err="$(git -C "$dir" checkout "$branch" 2>&1)"; then
      ok "$name: mudou → '$branch'"
    else
      echo -e "  ${RED}✗${NC} $name: checkout falhou (há mudanças não commitadas?)"
      printf '%s\n' "$err" | head -5 | sed 's/^/      /'
      return 0
    fi
  else
    ok "$name: já em '$branch'"
  fi

  if git -C "$dir" pull --ff-only --quiet 2>/dev/null; then
    ok "$name: sincronizado com origin"
  fi
}

# ── Branch switching ────────────────────────────────────────────────────────

if [[ "$BACK_BRANCH" == "$FRONT_BRANCH" ]]; then
  echo -e "${BLD}dev-checkout → '$BACK_BRANCH'${NC}"
else
  echo -e "${BLD}dev-checkout → back:'$BACK_BRANCH'  front:'$FRONT_BRANCH'${NC}"
fi

sec "Backend (System_X_System)"
checkout_repo "backend" "$BACKEND_DIR" "$BACK_BRANCH"

sec "Frontend (System_X_System_React)"
checkout_repo "frontend" "$FRONTEND_DIR" "$FRONT_BRANCH" "true"

# ── Servers ─────────────────────────────────────────────────────────────────

sec "Servidores"

backends_up=false
front_up=false
(port_in_use 5000 && port_in_use 8081) && backends_up=true || true
port_in_use 5173 && front_up=true || true

if $backends_up && $front_up; then
  ok "Todos rodando (:5000 :8081 :5173)"
  echo -e "  → Air recompila o backend; Vite HMR recarrega o frontend"
  echo ""
  exit 0
fi

if $backends_up; then
  ok "Backends rodando (:5000 :8081)"
else
  warn "Backends offline"
fi
if $front_up; then
  ok "Frontend rodando (:5173)"
else
  warn "Frontend offline"
fi

echo ""

if ! $backends_up && ! $front_up; then
  echo -e "  Iniciando tudo com ${BLD}make dev${NC}..."
  echo -e "  (Ctrl+C para parar todos os processos)"
  echo ""
  exec make -C "$PROJECT_DIR" dev

elif ! $backends_up && $front_up; then
  echo -e "  Iniciando backends com ${BLD}make run-dev${NC}..."
  echo -e "  (Ctrl+C para parar API e Game; Vite continua rodando)"
  echo ""
  exec make -C "$BACKEND_DIR" run-dev

elif $backends_up && ! $front_up; then
  echo -e "  Iniciando frontend com ${BLD}npm run dev${NC}..."
  echo -e "  (Ctrl+C para parar o Vite; backends continuam rodando)"
  echo ""
  exec npm --prefix "$FRONTEND_DIR" run dev
fi
