# apps/boopmaster/BUILD

files(
    name="files",
    sources=[
        "**/*",
        "!BUILD",
    ],
)

archive(
    name="dist",
    format="tar.gz",
    files=[":files"],
    output_path="apps/boopmaster.tar.gz",
)
