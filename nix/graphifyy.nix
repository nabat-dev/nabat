# graphifyy CLI with a fixed-output wheel closure (see graphifyy-wheel-lock.nix).
# Python 3.12 matches cp312 wheels for the core tree-sitter binding; grammars use abi3 wheels.

{ pkgs }:

let
  pyp = pkgs.python312Packages;
  wheelLock = import ./graphifyy-wheel-lock.nix;
  system = pkgs.stdenv.hostPlatform.system;

  fetchWheel =
    pname: version:
    let
      key = "${pname}@${version}";
      meta =
        wheelLock.${key}.${system} or (throw "graphifyy wheels: no entry for ${key} on ${system}");
    in
    pkgs.fetchurl {
      inherit (meta) url hash;
    };

  mkWheelPkg =
    {
      pname,
      version,
      propagatedBuildInputs ? [ ],
    }:
    pyp.buildPythonPackage {
      inherit pname version propagatedBuildInputs;
      format = "wheel";
      src = fetchWheel pname version;
      doCheck = false;
      nativeBuildInputs = with pyp; [ pip ];
    };

  networkx = mkWheelPkg {
    pname = "networkx";
    version = "3.6.1";
  };

  treeSitter = mkWheelPkg {
    pname = "tree-sitter";
    version = "0.25.2";
  };

  langSpecs = [
    {
      pname = "tree-sitter-c";
      version = "0.24.2";
    }
    {
      pname = "tree-sitter-cpp";
      version = "0.23.4";
    }
    {
      pname = "tree-sitter-c-sharp";
      version = "0.23.5";
    }
    {
      pname = "tree-sitter-elixir";
      version = "0.3.5";
    }
    {
      pname = "tree-sitter-go";
      version = "0.25.0";
    }
    {
      pname = "tree-sitter-java";
      version = "0.23.5";
    }
    {
      pname = "tree-sitter-javascript";
      version = "0.25.0";
    }
    {
      pname = "tree-sitter-julia";
      version = "0.23.1";
    }
    {
      pname = "tree-sitter-kotlin";
      version = "1.1.0";
    }
    {
      pname = "tree-sitter-lua";
      version = "0.5.0";
    }
    {
      pname = "tree-sitter-objc";
      version = "3.0.2";
    }
    {
      pname = "tree-sitter-php";
      version = "0.24.1";
    }
    {
      pname = "tree-sitter-powershell";
      version = "0.26.3";
    }
    {
      pname = "tree-sitter-python";
      version = "0.25.0";
    }
    {
      pname = "tree-sitter-ruby";
      version = "0.23.1";
    }
    {
      pname = "tree-sitter-rust";
      version = "0.24.2";
    }
    {
      pname = "tree-sitter-scala";
      version = "0.26.0";
    }
    {
      pname = "tree-sitter-swift";
      version = "0.0.1";
    }
    {
      pname = "tree-sitter-typescript";
      version = "0.23.2";
    }
    {
      pname = "tree-sitter-verilog";
      version = "1.0.3";
    }
    {
      pname = "tree-sitter-zig";
      version = "1.1.2";
    }
  ];

  langs = map (
    spec:
    mkWheelPkg {
      pname = spec.pname;
      version = spec.version;
      propagatedBuildInputs = [ treeSitter ];
    }
  ) langSpecs;

  graphifyApp = pyp.buildPythonApplication rec {
    pname = "graphifyy";
    version = "0.6.7";
    format = "wheel";
    src = fetchWheel pname version;
    propagatedBuildInputs = [
      networkx
      treeSitter
    ] ++ langs;
    doCheck = false;
    nativeBuildInputs = with pyp; [ pip ];
    meta = {
      mainProgram = "graphify";
      description = "graphify CLI from PyPI graphifyy (${version}), pinned wheel closure";
    };
  };
in
{
  inherit graphifyApp;
}
