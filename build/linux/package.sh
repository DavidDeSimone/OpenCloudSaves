flatpak uninstall --user org.github.opencloudsaves.opencloudsaves
flatpak-builder --force-clean --user --disable-rofiles-fuse --repo=repo build-dir org.github.opencloudsaves.opencloudsaves.yml
#flatpak --user remote-add --if-not-exists --no-gpg-verify tutorial-repo repo
flatpak  install --user tutorial-repo org.github.opencloudsaves.opencloudsaves