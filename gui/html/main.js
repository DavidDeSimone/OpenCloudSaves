const DEFAULT_SEARCH_SCORE = 150;
const BIG_SYNC_SIZE = 100; // Megabytes
let pendingEdit = null;

async function onSelectClicked(element, name) {
    const dir = await openDirDialog();
    pathEl = document.getElementById(`${name}-path`);
    pathEl.value = dir;
}

async function onAddPathClicked(platform) {
    log(`Adding a path to ${platform}`);
    const containerEl = document.getElementById(`${platform}-container`);
    const copyContainer = containerEl.cloneNode(true);
    containerEl.appendChild(copyContainer);
}


async function onSyncButtonSuccess(element, name) {
    await setupSyncModal(name);
    const confirmEl = document.getElementById('bisync-confirm');
    confirmEl.style.display = 'block';
}

async function onSyncButtonClicked(element, name) {
    const sizeEl = document.getElementById(`${name}-total-size`);
    const size = sizeEl !== null ? parseInt(sizeEl.innerText) : 0;
    const shouldNotPrompt = await getShouldNotPromptForLargeSyncs();
    if (!shouldNotPrompt && size >= BIG_SYNC_SIZE) {
        makeConfirmationPopup({
            title: `Please Confirm Large Sync`,
            subtitle: `You are attempting to sync ${size}MB of data. Please confirm if you would like to continue. If this number does not look correct, please check your save file definitions via the "Edit" button. This sync may take a long time. You can disable seeing this warning in settings.`,
            onConfirm: () => {
                onSyncButtonSuccess(element, name);
            }
        })
    } else {
        await onSyncButtonSuccess(element, name);
    }
}

async function onEditButtonClicked(element, name) {
    await log(`Edit ${name}`);
    pendingEdit = name;

    const def = await fetchGamedef(name);
    await log(JSON.stringify(def));
    deserGamedef(def);
    await log("Deser Gamedef....");
    openAddGamesMenu(false);
}

function onRemoveButtonClicked(element, name) {
    log(`Remove ${name}`)
    removeGamedefByKey(name)
    refresh()
}

function onAddGameClosed() {
    pendingEdit = null;
    document.getElementById('id01').style.display='none';
    document.getElementById('accordion-cont').style.display='block';
    refresh();
}

function openAddGamesMenu(deser = true) {
    document.getElementById('accordion-cont').style.display='none';
    document.getElementById('id01').style.display='block';
    if (deser) {
        deserGamedef({
            Name: "New Game"
        })
    }
}

function onClearUserSettings() {
    clearUserSettings();
    refresh();
}

function deserGamedef(gamedef) {
    const gamenameEl = document.getElementById('gamename');
    gamenameEl.value = gamedef.Name;

    const flags = document.getElementById('flags');
    flags.value = gamedef.CustomFlags || "";

    ["Windows", "MacOS", "Linux"].forEach(element => {
        const def = gamedef[element];
        if (!def) {
            return;
        }

        // The html already has an element, so we will only
        // add an element starting from index 1
        for (let i = 1; i < def.length; ++i) {
            onAddPathClicked(element)
        }

        pathElems = document.getElementsByClassName(`${element}-path`)
        includeElms = document.getElementsByClassName(`${element}-include`)

        for (let i = 0; i < pathElems.length; ++i) {
            const dataPath = def[i];
            if (!dataPath) {
                continue;
            }
            log(JSON.stringify(dataPath))

            pathEl = pathElems[i];
            includeEl = includeElms[i];

            includeEl.value = dataPath.Include || "";
            pathEl.value = dataPath.Path;

        }
    });
}

async function submitGamedef() {
    document.getElementById('id01').style.display='none';
    gamenameEl = document.getElementById('gamename');
    const flags = document.getElementById('flags').value || "";
    let result = {
        Name: gamenameEl.value,
        Windows: [],
        MacOS: [],
        Linux: [],
        CustomFlags: flags,
    };

    ["Windows", "MacOS", "Linux"].forEach(element => {
        pathElems = document.getElementsByClassName(`${element}-path`)
        includeElms = document.getElementsByClassName(`${element}-include`)

        for (let i = 0; i < pathElems.length; ++i) {
            pathEl = pathElems[i];
            includeEl = includeElms[i];
    
            result[element].push({
                Path: pathEl.value || "",
                Include: includeEl.value || "",
            });
        }
    });

    if (pendingEdit) {
        await removeGamedefByKey(pendingEdit);
        pendingEdit = null;
    }

    await commitGamedef(result)
    refresh()
}

async function onChangeSearch(element) {
    const name = element.value

    var acc = document.getElementsByClassName("accordion");
    var i;
    
    for (i = 0; i < acc.length; i++) {
        if (name === "") {
            acc[i].style.display = "block";
            continue
        }

        const res = fuzzyMatch(name.toLowerCase(), acc[i].id.replace("-accordion", "").toLowerCase())
        if (res[0] || res[1] > DEFAULT_SEARCH_SCORE) {
            acc[i].style.display = "block";
        } else {
            acc[i].style.display = "none";
        }
    }
}

async function onSetCloudClicked(element) {
    setCloudSelectScreen();
}

async function onSyncSettingsClicked(element) {
    onSettingsModalOpen()
}

function setupAccordionHandler() {
    var acc = document.getElementsByClassName("accordion");
    var i;
    
    for (i = 0; i < acc.length; i++) {
      acc[i].addEventListener("click", function() {
        this.classList.toggle("active");
        var panel = this.nextElementSibling;
        if (panel.style.display === "block") {
          panel.style.display = "none";
        } else {
          panel.style.display = "block";
        }
      });
    }
}

setTimeout(async () => { 
    setupAccordionHandler();
    await require('html/fuzzy-search.js');
});

