# Copyright 2018 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Genrule wrapper around the go-bindata utility.
# forked from https://github.com/kubernetes/kubernetes/blob/master/build/bindata.bzl
# this variant only supports once source dir
# IMPORTANT: Any changes to this rule may also require changes to hack/generate.sh.
def go_bindata(
        # these are for bazel
        name,
        srcs,
        outs,
        # -prefix flag
        prefix,
        # the sources to pass to go-bindata
        # NOTE: these must be contained in srcs or bazel will not populate them
        bindata_srcs,
        # label for the file containing the go:generate directive, so we can
        # run go-bindata from the containing package / directory
        generate_directive_file,
        # -mode flag
        mode = "0666",
        # -nometadata flag
        include_metadata = True,
        # -pkg flag
        pkg = "generated",
        # -ignore flag
        ignores = [],
        **kw):
    args = ['-pkg', pkg]
    if not include_metadata:
        args.append("-nometadata")
    if mode:
        args.append("-mode=%s" % mode)
    for ignore in ignores:
        args.extend(["-ignore", "'%s'" % ignore])
    if prefix:
        args.append("-prefix=%s" % prefix)

    native.genrule(
        name = name,
        srcs = srcs,
        outs = outs,
        cmd = " ".join([
            # save the original working directory (repo root)
            # all paths will be relative to this
            "ORIG_WD=$$(pwd);",
            # cd to the directory containing the go:generate directive,
            # since this is what go generate would do
            "cd $$(dirname $(location %s));" % (generate_directive_file),
            # run go-bindata
            "$$ORIG_WD/$(location %s) -o $$ORIG_WD/$@ %s %s" % (
                "//vendor/github.com/jteeuwen/go-bindata/go-bindata:go-bindata",
                " ".join(args), 
                bindata_srcs,
            ),
        ]),
        tools = [
            "//vendor/github.com/jteeuwen/go-bindata/go-bindata",
            generate_directive_file,
        ],
        **kw
    )
