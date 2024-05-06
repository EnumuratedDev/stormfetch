command_exists() {
    if [ -z "$1" ]; then
        return 1
    fi
    if command -v "$1" &> /dev/null; then
        return 0
    else
        return 1
    fi
}

get_shell() {
    case ${SHELL##*/} in
    "")
        echo "Unknown"
        ;;
    sh|ash|dash|es)
        echo "${SHELL##*/} $(${SHELL##*/} --version)"
        ;;
    bash)
        echo "${SHELL##*/} $(${SHELL##*/} -c "echo "'$BASH_VERSION')"
        ;;
    *)
        SHELL_NAME=${SHELL##*/}
        SHELL_VERSION="$($SHELL --version)"
        SHELL_VERSION=$(SHELL_VERSION//,//)
        SHELL_VERSION=$(SHELL_VERSION// //)
        SHELL_VERSION=$(SHELL_VERSION//VERSION//)
        SHELL_VERSION=$(SHELL_VERSION//"$SHELL_NAME"//)
        echo "$SHELL_NAME $SHELL_VERSION"
        unset SHELL_NAME
        unset SHELL_VERSION
        ;;
    esac
}

get_cpu_name() {
    grep -m1 "model name" /proc/cpuinfo | cut -d: -f2 | xargs
}

get_total_mem() {
     free --mebi -t | grep 'Total' | tr -s ' ' | cut -d" " -f2
}

get_free_mem() {
    free --mebi -t | grep 'Total' | tr -s ' ' | cut -d" " -f3
}

get_used_mem() {
    free --mebi -t | grep 'Total' | tr -s ' ' | cut -d" " -f4
}

get_de_wm() {
    if [[ $DESKTOP_SESSION == "plasma" ]]; then
      echo "KDE Plasma $(plasmashell --version | sed 's/plasmashell //g')"
    else
      echo "$DESKTOP_SESSION"
    fi
}

get_screen_resolution() {
    if xhost >& /dev/null ;then
      xdpyinfo | grep dimensions | tr -s ' ' | cut -d " " -f3
    fi
}

get_packages() {
    ARRAY=()
    if command_exists dpkg; then
        ARRAY+=("$(dpkg-query -f '.\n' -W | wc -l) (dpkg)")
    fi
    if command_exists pacman; then
        ARRAY+=("$(pacman -Q | wc -l) (pacman)")
    fi
    if command_exists rpm; then
        ARRAY+=("$(rpm -qa | wc -l) (rpm)")
    fi
    if command_exists bpm; then
        ARRAY+=("$(bpm list -n) (bpm)")
    fi
    if command_exists emerge; then
        ARRAY+=("$(ls -l /var/db/pkg/ | wc -l) (emerge)")
    fi
    if command_exists flatpak; then
        ARRAY+=("$(flatpak list | wc -l) (flatpak)")
    fi
    if command_exists snap; then
        ARRAY+=("$(snap list | wc -l) (snap)")
    fi
    echo "${ARRAY[@]}"
    unset ARRAY
}
