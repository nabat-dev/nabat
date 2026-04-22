#!/usr/bin/env bash
set -euo pipefail

PACKAGES=(
  "graphifyy|0.6.7"
  "networkx|3.6.1"
  "tree-sitter|0.25.2"
  "tree-sitter-c|0.24.2"
  "tree-sitter-cpp|0.23.4"
  "tree-sitter-c-sharp|0.23.5"
  "tree-sitter-elixir|0.3.5"
  "tree-sitter-go|0.25.0"
  "tree-sitter-java|0.23.5"
  "tree-sitter-javascript|0.25.0"
  "tree-sitter-julia|0.23.1"
  "tree-sitter-kotlin|1.1.0"
  "tree-sitter-lua|0.5.0"
  "tree-sitter-objc|3.0.2"
  "tree-sitter-php|0.24.1"
  "tree-sitter-powershell|0.26.3"
  "tree-sitter-python|0.25.0"
  "tree-sitter-ruby|0.23.1"
  "tree-sitter-rust|0.24.2"
  "tree-sitter-scala|0.26.0"
  "tree-sitter-swift|0.0.1"
  "tree-sitter-typescript|0.23.2"
  "tree-sitter-verilog|1.0.3"
  "tree-sitter-zig|1.1.2"
)

PY_TAG="cp312"

wheel_any() {
  jq -r '[.urls[] | select(.packagetype=="bdist_wheel") | select(.filename | test("py3-none-any\\.whl$"))] | .[0]'
}

wheel_tree_sitter_py() {
  local plat_re="$1"
  jq -r --arg py "$PY_TAG" --arg plat "$plat_re" '
    [.urls[] | select(.packagetype=="bdist_wheel")
     | select(.filename | test($py))
     | select(.filename | test($plat))]
    | sort_by(.filename | length)
    | .[0]
  '
}

wheel_abi3() {
  local plat_re="$1"
  jq -r --arg plat "$plat_re" '
    [.urls[] | select(.packagetype=="bdist_wheel")
     | select(.filename | test("abi3"))
     | select(.filename | test($plat))]
    | sort_by(.filename | length)
    | .[0]
  '
}

plat_regex_for_system() {
  case "$1" in
    x86_64-linux) echo 'manylinux.*x86_64' ;;
    aarch64-linux) echo 'manylinux.*aarch64' ;;
    x86_64-darwin) echo 'macosx.*x86_64' ;;
    aarch64-darwin) echo 'macosx.*arm64' ;;
    *) echo '' ;;
  esac
}

select_wheel_json() {
  local pkg="$1"
  local json="$2"
  local system="$3"

  local any
  any="$(echo "$json" | wheel_any)"
  if [[ "$any" != "null" ]]; then
    echo "$any"
    return
  fi

  local plat
  plat="$(plat_regex_for_system "$system")"
  local w
  if [[ "$pkg" == "tree-sitter" ]]; then
    w="$(echo "$json" | wheel_tree_sitter_py "$plat")"
  else
    w="$(echo "$json" | wheel_abi3 "$plat")"
  fi
  echo "$w"
}

echo "{"

first_pkg=1
for pv in "${PACKAGES[@]}"; do
  pkg="${pv%%|*}"
  ver="${pv##*|}"
  json="$(curl -fsSL "https://pypi.org/pypi/${pkg}/${ver}/json")"

  if [[ $first_pkg -eq 0 ]]; then
    echo
  fi
  first_pkg=0

  printf '  "%s@%s" = {\n' "$pkg" "$ver"

  first_sys=1
  for sys in x86_64-linux aarch64-linux x86_64-darwin aarch64-darwin; do
    wjson="$(select_wheel_json "$pkg" "$json" "$sys")"
    if [[ "$wjson" == "null" || -z "$wjson" ]]; then
      echo "missing wheel: $pkg $ver ($sys)" >&2
      exit 1
    fi
    url="$(echo "$wjson" | jq -r '.url')"
    hex="$(echo "$wjson" | jq -r '.digests.sha256')"
    if [[ -z "$url" || "$url" == "null" || -z "$hex" || "$hex" == "null" ]]; then
      echo "missing digest: $pkg $ver ($sys)" >&2
      exit 1
    fi
    sri="$(nix hash to-sri --type sha256 "$hex" 2>/dev/null)"
    if [[ $first_sys -eq 0 ]]; then
      echo
    fi
    first_sys=0
    printf '    %s = {\n      url = "%s";\n      hash = "%s";\n    };' "$sys" "$url" "$sri"
  done
  echo
  echo "  };"
done

echo "}"
