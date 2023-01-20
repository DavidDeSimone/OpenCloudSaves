flatpak uninstall --user org.github.opencloudsaves.opencloudsaves
flatpak-builder --force-clean --disable-rofiles-fuse --user --install-deps-from --repo=repo build-dir org.github.opencloudsaves.opencloudsaves.yml
flatpak --user remote-add --if-not-exists --no-gpg-verify tutorial-repo repo
flatpak --user install tutorial-repo org.github.opencloudsaves.opencloudsaves