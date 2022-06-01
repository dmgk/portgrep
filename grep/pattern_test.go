package grep

import (
	"testing"
)

func testStringPattern(t *testing.T, pat *stringPattern, val string, isRegexp bool, matches []string, nomatches []string) {
	pat.val = val
	var r *Regexp
	var err error
	if isRegexp {
		r, err = pat.CompileNoQuote(0, 0)
	} else {
		r, err = pat.Compile(0, 0)
	}
	if err != nil {
		t.Fatal(err)
	}

	for i, x := range matches {
		res, err := r.Match([]byte(x))
		if err != nil {
			t.Fatal(err)
		}
		if res == nil {
			t.Errorf("[matches #%d] expected %q to match %q", i, val, x)
		}
	}

	for i, x := range nomatches {
		res, err := r.Match([]byte(x))
		if err != nil {
			t.Fatal(err)
		}
		if res != nil {
			t.Errorf("[nomatches #%d] expected %q to not match %q, got %#v", i, val, x, res)
		}
	}
}

func testBoolPattern(t *testing.T, pat *boolPattern, matches []string, nomatches []string) {
	pat.val = true
	r, err := pat.Compile(0, 0)
	if err != nil {
		t.Fatal(err)
	}

	for i, x := range matches {
		res, err := r.Match([]byte(x))
		if err != nil {
			t.Fatal(err)
		}
		if res == nil {
			t.Errorf("[matches #%d] expected to match %q", i, x)
		}
	}

	for i, x := range nomatches {
		res, err := r.Match([]byte(x))
		if err != nil {
			t.Fatal(err)
		}
		if res != nil {
			t.Errorf("[nomatches #%d] expected to not match %q, got %#v", i, x, res)
		}
	}
}

func TestBroken(t *testing.T) {
	matches := []string{
		"BROKEN=	doesn't build",
		"BROKEN_i386=	broken",
	}

	nomatches := []string{
		"BROKEN_=	doesn't build",
		"_BROKEN=	doesn't build",
	}

	testBoolPattern(t, broken, matches, nomatches)
}

func TestDepends(t *testing.T) {
	matches := []string{
		"BUILD_DEPENDS=	dash:shells/dash bash:shells/bash",
		"BUILD_DEPENDS?=	dash:shells/dash bash:shells/bash",
		"BUILD_DEPENDS+=	dash:shells/dash bash:shells/bash",
		"RUN_DEPENDS=	bash>0:shells/bash dash:shells/dash",
		"OPT_DEPENDS=	/usr/local/bin/bash:shells/bash",
	}

	nomatches := []string{
		"BUILD_DEPENDS=	bash-devel:/shells/bash-devel",
		"BUILD_DEPENDS=	other-bash:/shells/other-bash",
		"_DEPENDS=		bash:/shells/bash",
	}

	testStringPattern(t, depends, "bash", false, matches, nomatches)
}

func TestBuildDepends(t *testing.T) {
	matches := []string{
		"BUILD_DEPENDS=	dash:shells/dash bash:shells/bash",
		"OPT_BUILD_DEPENDS=	bash:shells/bash",
	}

	nomatches := []string{
		"RUN_DEPENDS=	bash-devel:/shells/bash-devel",
	}

	testStringPattern(t, buildDepends, "bash", false, matches, nomatches)
}

func TestLibDepends(t *testing.T) {
	matches := []string{
		"LIB_DEPENDS=	dash:shells/dash bash:shells/bash",
		"LIB_DEPENDS?=	dash:shells/dash bash:shells/bash",
		"LIB_DEPENDS+=	dash:shells/dash bash:shells/bash",
		"OPT_LIB_DEPENDS=	bash:shells/bash",
	}

	nomatches := []string{
		"RUN_DEPENDS=	bash-devel:/shells/bash-devel",
	}

	testStringPattern(t, libDepends, "bash", false, matches, nomatches)
}

func TestRunDepends(t *testing.T) {
	matches := []string{
		"RUN_DEPENDS=	dash:shells/dash bash:shells/bash",
		"RUN_DEPENDS?=	dash:shells/dash bash:shells/bash",
		"RUN_DEPENDS+=	dash:shells/dash bash:shells/bash",
		"OPT_RUN_DEPENDS=	bash:shells/bash",
	}

	nomatches := []string{
		"BUILD_DEPENDS=	bash-devel:/shells/bash-devel",
	}

	testStringPattern(t, runDepends, "bash", false, matches, nomatches)
}

func TestOnlyForArchs(t *testing.T) {
	matches := []string{
		"ONLY_FOR_ARCHS=	amd64 i386",
	}

	nomatches := []string{
		"ONLY_FOR_ARCHS=	armv7",
	}

	testStringPattern(t, onlyForArchs, "amd64", false, matches, nomatches)
}

func TestMaintainer(t *testing.T) {
	matches := []string{
		"MAINTAINER=ports@freebsd.org",
		"MAINTAINER=	ports@FreeBSD.org",
		"MAINTAINER=	ports@freebsd",
		" MAINTAINER=	ports@freebsd",
	}

	nomatches := []string{
		"MAINTAINER=xports@freebsd.org",
		"MAINTAINER=	xports@freebsd.org",
		"XMAINTAINER=	ports@freebsd.org",
		"_MAINTAINER=	ports@freebsd.org",
	}

	testStringPattern(t, maintainer, "ports@", false, matches, nomatches)
}

func TestPortname(t *testing.T) {
	matches := []string{
		"PORTNAME=	modules2tuple",
		"PORTNAME=	modules",
		"PORTNAME=	modules-blah",
		" PORTNAME=modules",
	}

	nomatches := []string{
		"PORTNAME=	pam-modules",
		"PORTNAME=	-modules",
		"XPORTNAME=	modules",
	}

	testStringPattern(t, portname, "modules", false, matches, nomatches)
}

func TestMaintainerRe(t *testing.T) {
	matches := []string{
		"MAINTAINER=	ports@freebsd.org",
		"MAINTAINER=	ports@FreeBSD.org",
	}

	nomatches := []string{
		"MAINTAINER=	xports@freebsd.org",
		"MAINTAINER=	dmgk@freebsd.org",
	}

	testStringPattern(t, maintainer, "p.*s@", true, matches, nomatches)
}

func TestUses(t *testing.T) {
	matches := []string{
		"USES=go",
		"USES=	go",
		"USES=	go xorg",
		"USES=	gmake go xorg",
		"USES=	gmake go",
		"VAR_USES=	go",
	}

	nomatches := []string{
		"USES=	cargo",
		"USE=	go",
		"USES=	gmake golang",
		"USES=	gmake-go xorg",
		"USES=	go-test",
		"USES=	go.test",
		"XUSES=	go",
	}

	testStringPattern(t, uses, "go", false, matches, nomatches)
}

func TestPlistRe(t *testing.T) {
	matches := []string{
		"PLIST_FILES=	bin/bash",
		"PLIST_FILES =bin/bash",
		"PLIST_FILES =	bin/bash",
		"OPT_PLIST_FILES =	bin/bash",
	}

	nomatches := []string{
		"PLIST_FILES=	bin/zsh",
	}

	testStringPattern(t, plist, "bash", false, matches, nomatches)
}
