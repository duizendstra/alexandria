// Package buildstamp records and verifies which build produced a binary.
//
// BC:      Platform
// Concern: Is the binary in front of me the code I think it is?
//
// When an operator runs a tool that changes production state, "which build was
// that?" is not a curiosity — it is the difference between an incident you can
// reconstruct and one you cannot. A stamp answers it: the commit, whether the
// working tree was clean, when it was built, and the provenance of the
// dependencies that were not pinned by the module graph.
//
// The package does two things:
//
//   - Get reads the stamp of the running binary, from -ldflags values when the
//     build script sets them and from Go's embedded VCS settings otherwise.
//   - ParseStamp reads a stamp back out of a version line, so a wrapper script
//     or a supervising process can verify a binary it did not build. Matches
//     then applies the rules that make a stamp trustworthy.
//
// Those rules are deliberately strict, because each loose end has caused a
// real incident somewhere:
//
//   - the commit must be a full 40-character SHA and match the expected one.
//     A short or missing SHA is refused rather than assumed;
//   - "unknown" is never acceptable. It is what you get when the build lost
//     its VCS context, which is exactly when you least want to guess;
//   - a dirty working tree is refused. Uncommitted changes cannot be
//     reconstructed later, so a dirty build is unreproducible by definition;
//   - recorded dependency stamps must be clean too. A pinned binary built
//     against a dirty local dependency is not pinned.
//
// Zero dependencies outside the standard library.
package buildstamp
