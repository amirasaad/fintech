#!/usr/bin/env bash
# Requires Bash 4+ for associative arrays.

# 1. Parse app/app.go for bus.Register lines
declare -A handler_to_event
declare -A event_to_handlers

while read -r line; do
  if [[ $line =~ bus\.Register\("([A-Za-z0-9]+)"[[:space:]]*,[[:space:]]*([a-zA-Z0-9_.]+)\( ]]; then
    event_type="${BASH_REMATCH[1]}"
    handler_func="${BASH_REMATCH[2]}"
    handler_func="${handler_func##*.}" # strip package prefix
    handler_to_event["$handler_func"]="$event_type"
    event_to_handlers["$event_type"]+="$handler_func "
  fi
done < app/app.go

# 2. Parse handler files for bus.Emit lines
declare -A handler_emits

while IFS= read -r -d '' file; do
  current_handler=""
  while read -r line; do
    if [[ $line =~ ^func[[:space:]]+([A-Za-z0-9_]+)\( ]]; then
      current_handler="${BASH_REMATCH[1]}"
    fi
    if [[ $line =~ bus\.Emit\([^,]+,[[:space:]]*(?:[a-zA-Z0-9_]+\.)*([A-Za-z0-9]+Event)[[:space:]]*\{ ]]; then
      emitted_event="${BASH_REMATCH[1]}"
      if [[ -n $current_handler ]]; then
        handler_emits["$current_handler"]+="$emitted_event "
      fi
    fi
  done < "$file"
done < <(find pkg/handler -type f -name '*.go' -print0)

# 3. Build event-to-event graph
declare -A graph
for event_type in "${!event_to_handlers[@]}"; do
  for handler in ${event_to_handlers[$event_type]}; do
    for emitted in ${handler_emits[$handler]}; do
      graph["$event_type"]+="$emitted "
    done
  done
done

# 4. Print event flow graph
echo -e "\nEvent Flow Graph:"
for from in "${!graph[@]}"; do
  has_edges=0
  for to in ${graph[$from]}; do
    echo "  $from -> $to"
    has_edges=1
  done
  if [[ $has_edges -eq 0 ]]; then
    echo "  $from -> []"
  fi
done

# 5. Cycle detection (DFS)
declare -A visited
declare -A stack
has_cycle=0

# shellcheck disable=SC2034
path=()

dfs() {
  local node="$1"
  stack["$node"]=1
  visited["$node"]=1
  path+=("$node")
  for neighbor in ${graph[$node]}; do
    if [[ ${stack[$neighbor]} -eq 1 ]]; then
      echo "Cycle detected: ${path[*]} $neighbor"
      has_cycle=1
      return 0
    fi
    if [[ -z ${visited[$neighbor]} ]]; then
      dfs "$neighbor"
    fi
  done
  stack["$node"]=0
  unset 'path[${#path[@]}-1]'
}

echo -e "\nCycle Detection:"
for node in "${!graph[@]}"; do
  if [[ -z ${visited[$node]} ]]; then
    path=()
    dfs "$node"
  fi
done

if [[ $has_cycle -eq 1 ]]; then
  echo -e "\n❌ Event cycle(s) detected! Review your event flow."
  exit 1
else
  echo -e "\n✅ No event cycles detected."
  exit 0
fi
