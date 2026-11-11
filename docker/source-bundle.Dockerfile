# Carrier image for the source backup published by the CD workflow: it only holds a git bundle
# with the full history of the repository at release time.
#
# Restore with:
#   docker create --name src <image>:<tag> x
#   docker cp src:/source.bundle . && docker rm src
#   git clone source.bundle verana-node
FROM scratch
COPY source.bundle /source.bundle
