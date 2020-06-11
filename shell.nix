{ pkgs ? (import ./default.nix).nixpkgs {
	config = {
		packageOverrides = pkgs: {
    	leveldb = pkgs.callPackage ./leveldb_with_snappy.nix {};
	  };
	};
} }:

with pkgs;
{ kythe-compile = stdenv.mkDerivation {
    name = "kythe-compile";
    buildInputs = [
      bazel cmake zlib asciidoc sourceHighlight libuuid.dev ncurses.dev jdk
      # TODO(robinp): pull in with nixpkgs? now bazel-build works if triggered
      # from nix-shell, but a pure nix-build would fail?
      graphviz python go coreutils
      # with our snappy fix, see https://github.com/NixOS/nixpkgs/issues/89850
      leveldb
    ];
    shellHook = ''
      echo === Generating .bazelrc.nix fragment.
      sh gen-bazelrc-nix.sh
    '';
};
}
