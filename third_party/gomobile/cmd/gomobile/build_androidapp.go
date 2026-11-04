// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"crypto/x509"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/mobile/internal/binres"
	"golang.org/x/tools/go/packages"
)

func goAndroidBuild(pkg *packages.Package, targets []targetInfo) (map[string]bool, error) {
	ndkRoot, err := ndkRoot(targets...)
	if err != nil {
		return nil, err
	}
	appName := path.Base(pkg.PkgPath)
	libName := androidPkgName(appName)

	// TODO(hajimehoshi): This works only with Go tools that assume all source files are in one directory.
	// Fix this to work with other Go tools.
	dir := filepath.Dir(pkg.GoFiles[0])

	manifestPath := filepath.Join(dir, "AndroidManifest.xml")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		buf := new(bytes.Buffer)
		buf.WriteString(`<?xml version="1.0" encoding="utf-8"?>`)
		err := manifestTmpl.Execute(buf, manifestTmplData{
			// TODO(crawshaw): a better package path.
			JavaPkgPath: "org.golang.todo." + libName,
			Name:        strings.Title(appName),
			LibName:     libName,
		})
		if err != nil {
			return nil, err
		}
		manifestData = buf.Bytes()
		if buildV {
			fmt.Fprintf(os.Stderr, "generated AndroidManifest.xml:\n%s\n", manifestData)
		}
	} else {
		libName, err = manifestLibName(manifestData)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %v", manifestPath, err)
		}
	}

	libFiles := []string{}
	nmpkgs := make(map[string]map[string]bool) // map: arch -> extractPkgs' output

	for _, t := range targets {
		toolchain := ndk.Toolchain(t.arch)
		libPath := "lib/" + toolchain.abi + "/lib" + libName + ".so"
		libAbsPath := filepath.Join(tmpdir, libPath)
		if err := mkdir(filepath.Dir(libAbsPath)); err != nil {
			return nil, err
		}
		err = goBuild(
			pkg.PkgPath,
			androidEnv[t.arch],
			"-buildmode=c-shared",
			"-o", libAbsPath,
		)
		if err != nil {
			return nil, err
		}
		nmpkgs[t.arch], err = extractPkgs(toolchain.Path(ndkRoot, "nm"), libAbsPath)
		if err != nil {
			return nil, err
		}
		libFiles = append(libFiles, libPath)
	}

	block, _ := pem.Decode([]byte(debugCert))
	if block == nil {
		return nil, errors.New("no debug cert")
	}
	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	if buildO == "" {
		buildO = androidPkgName(path.Base(pkg.PkgPath)) + ".apk"
	}
	if !strings.HasSuffix(buildO, ".apk") {
		return nil, fmt.Errorf("output file name %q does not end in '.apk'", buildO)
	}
	var out io.Writer
	if !buildN {
		f, err := os.Create(buildO)
		if err != nil {
			return nil, err
		}
		defer func() {
			if cerr := f.Close(); err == nil {
				err = cerr
			}
		}()
		out = f
	}

	var apkw *Writer
	if !buildN {
		apkw = NewWriter(out, privKey)
	}
	apkwCreate := func(name string) (io.Writer, error) {
		if buildV {
			fmt.Fprintf(os.Stderr, "apk: %s\n", name)
		}
		if buildN {
			return io.Discard, nil
		}
		return apkw.Create(name)
	}
	apkwWriteFile := func(dst, src string) error {
		w, err := apkwCreate(dst)
		if err != nil {
			return err
		}
		if !buildN {
			f, err := os.Open(src)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := io.Copy(w, f); err != nil {
				return err
			}
		}
		return nil
	}

	w, err := apkwCreate("classes.dex")
	if err != nil {
		return nil, err
	}
	dexData, err := base64.StdEncoding.DecodeString(dexStr)
	if err != nil {
		log.Fatalf("internal error bad dexStr: %v", err)
	}
	if _, err := w.Write(dexData); err != nil {
		return nil, err
	}

	for _, libFile := range libFiles {
		if err := apkwWriteFile(libFile, filepath.Join(tmpdir, libFile)); err != nil {
			return nil, err
		}
	}

	for _, t := range targets {
		toolchain := ndk.Toolchain(t.arch)
		if nmpkgs[t.arch]["golang.org/x/mobile/exp/audio/al"] {
			dst := "lib/" + toolchain.abi + "/libopenal.so"
			src := filepath.Join(gomobilepath, dst)
			if _, err := os.Stat(src); err != nil {
				return nil, errors.New("the Android requires the golang.org/x/mobile/exp/audio/al, but the OpenAL libraries was not found. Please run gomobile init with the -openal flag pointing to an OpenAL source directory.")
			}
			if err := apkwWriteFile(dst, src); err != nil {
				return nil, err
			}
		}
	}

	// Add any assets.
	var arsc struct {
		iconPath string
	}
	assetsDir := filepath.Join(dir, "assets")
	assetsDirExists := true
	fi, err := os.Stat(assetsDir)
	if err != nil {
		if os.IsNotExist(err) {
			assetsDirExists = false
		} else {
			return nil, err
		}
	} else {
		assetsDirExists = fi.IsDir()
	}
	if assetsDirExists {
		// if assets is a symlink, follow the symlink.
		assetsDir, err = filepath.EvalSymlinks(assetsDir)
		if err != nil {
			return nil, err
		}
		err = filepath.Walk(assetsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if name := filepath.Base(path); strings.HasPrefix(name, ".") {
				// Do not include the hidden files.
				return nil
			}
			if info.IsDir() {
				return nil
			}

			if rel, err := filepath.Rel(assetsDir, path); rel == "icon.png" && err == nil {
				arsc.iconPath = path
				// TODO returning here does not write the assets/icon.png to the final assets output,
				// making it unavailable via the assets API. Should the file be duplicated into assets
				// or should assets API be able to retrieve files from the generated resource table?
				return nil
			}

			name := "assets/" + path[len(assetsDir)+1:]
			return apkwWriteFile(name, path)
		})
		if err != nil {
			return nil, fmt.Errorf("asset %v", err)
		}
	}

	bxml, err := binres.UnmarshalXML(bytes.NewReader(manifestData), arsc.iconPath != "")
	if err != nil {
		return nil, err
	}

	// generate resources.arsc identifying single xxxhdpi icon resource.
	if arsc.iconPath != "" {
		pkgname, err := bxml.RawValueByName("manifest", xml.Name{Local: "package"})
		if err != nil {
			return nil, err
		}
		tbl, name := binres.NewMipmapTable(pkgname)
		if err := apkwWriteFile(name, arsc.iconPath); err != nil {
			return nil, err
		}
		w, err := apkwCreate("resources.arsc")
		if err != nil {
			return nil, err
		}
		bin, err := tbl.MarshalBinary()
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(bin); err != nil {
			return nil, err
		}
	}

	w, err = apkwCreate("AndroidManifest.xml")
	if err != nil {
		return nil, err
	}
	bin, err := bxml.MarshalBinary()
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(bin); err != nil {
		return nil, err
	}

	// TODO: add gdbserver to apk?

	if !buildN {
		if err := apkw.Close(); err != nil {
			return nil, err
		}
	}

	// TODO: return nmpkgs
	return nmpkgs[targets[0].arch], nil
}

// androidPkgName sanitizes the go package name to be acceptable as a android
// package name part. The android package name convention is similar to the
// java package name convention described in
// https://docs.oracle.com/javase/specs/jls/se8/html/jls-6.html#jls-6.5.3.1
// but not exactly same.
func androidPkgName(name string) string {
	var res []rune
	for _, r := range name {
		switch {
		case 'a' <= r && r <= 'z', 'A' <= r && r <= 'Z', '0' <= r && r <= '9':
			res = append(res, r)
		default:
			res = append(res, '_')
		}
	}
	if len(res) == 0 || res[0] == '_' || ('0' <= res[0] && res[0] <= '9') {
		// Android does not seem to allow the package part starting with _.
		res = append([]rune{'g', 'o'}, res...)
	}
	s := string(res)
	// Look for Java keywords that are not Go keywords, and avoid using
	// them as a package name.
	//
	// This is not a problem for normal Go identifiers as we only expose
	// exported symbols. The upper case first letter saves everything
	// from accidentally matching except for the package name.
	//
	// Note that basic type names (like int) are not keywords in Go.
	switch s {
	case "abstract", "assert", "boolean", "byte", "catch", "char", "class",
		"do", "double", "enum", "extends", "final", "finally", "float",
		"implements", "instanceof", "int", "long", "native", "private",
		"protected", "public", "short", "static", "strictfp", "super",
		"synchronized", "this", "throw", "throws", "transient", "try",
		"void", "volatile", "while":
		s += "_"
	}
	return s
}

// A random uninteresting private key.
// Must be consistent across builds so newer app versions can be installed.
// silikbox: do not distribute a shared private debug key.
var debugCert = func() string {
 key, err := rsa.GenerateKey(rand.Reader, 2048)
 if err != nil { panic(err) }
 return string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
}()
