{ pkgs ? import <nixpkgs> {} }:

with pkgs;
{ kythe-compile = stdenv.mkDerivation {
    name = "kythe-compile";
    buildInputs = [ bazel cmake zlib asciidoc sourceHighlight libuuid.dev ncurses.dev jdk ];
};
}