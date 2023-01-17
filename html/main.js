const DEFAULT_SEARCH_SCORE = 150;
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

    // TEST CODE
    // const containers = document.getElementsByClassName(`${platform}-path`);
    // for (let i = 0; i < containers.length; i++) {
    //     log(`Paths: ${containers[i].value}`);
    // }
}


async function onSyncButtonClicked(element, name) {
    const syncbtn = document.getElementById(`${name}-syncbtn`);
    const editbtn = document.getElementById(`${name}-editbtn`);
    const removebtn = document.getElementById(`${name}-removebtn`);

    syncbtn.disabled = true;
    editbtn.disabled = true;
    removebtn.disabled = true;

    log(`Sync ${name}`)
    await syncGame(name)
    const interval = setInterval(async () => {
        const logEl = document.getElementById(`${name}-log`);
        const progressEl = document.getElementById(`${name}-progress`);

        logEl.style.display = "block";
        progressEl.style.display = "block";

        const logValue = await pollLogs(name);
        if (logValue != "") {
            logEl.textContent = logValue
        }



        const res = await pollProgress(name);
        if (res.Total == 0) {
            return;
        }

        progressEl.style.width = `${(res.Current / res.Total) * 100}%`;
        if (res.Current == res.Total) {
            clearInterval(interval);
            syncbtn.disabled = false;
            editbtn.disabled = false;
            removebtn.disabled = false;
            setTimeout(() => {
                logEl.style.display = "none";
                progressEl.style.display = "none";
            }, 5000)
        }
    }, 1000)
}

async function onEditButtonClicked(element, name) {
    log(`Edit ${name}`);
    pendingEdit = name;

    const def = await fetchGamedef(name);
    deserGamedef(def);
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
    refresh();
}

function openAddGamesMenu(deser = true) {
    document.getElementById('id01').style.display='block';
    if (deser) {
        deserGamedef({
            Name: "New Game"
        })
    }
}

function deserGamedef(gamedef) {
    const gamenameEl = document.getElementById('gamename');
    gamenameEl.value = gamedef.Name;

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
        extElems = document.getElementsByClassName(`${element}-ext`)
        ignoreElems = document.getElementsByClassName(`${element}-ignore`)
        downloadElems = document.getElementsByClassName(`${element}-download`)
        uploadElems = document.getElementsByClassName(`${element}-upload`)
        deleteElems = document.getElementsByClassName(`${element}-delete`)

        for (let i = 0; i < pathElems.length; ++i) {
            const dataPath = def[i];
            if (!dataPath) {
                continue;
            }
            log(JSON.stringify(dataPath))

            pathEl = pathElems[i];
            extEl = extElems[i];
            ignoreEl = ignoreElems[i];
            downloadEl = downloadElems[i];
            uploadEl = uploadElems[i];
            deleteEl = deleteElems[i];

            extEl.value = dataPath.Exts != null && dataPath.Exts.length > 0 ? dataPath.Exts.join(',') : "";
            ignoreEl.value = dataPath.Ignore != null && dataPath.Ignore.length > 0 ? dataPath.Ignore.join(',') : "";
            pathEl.value = dataPath.Path;
            downloadEl.checked = dataPath.Download;
            uploadEl.checked = dataPath.Upload;
            deleteEl.checked = dataPath.Delete;
        }
    });
}

function submitGamedef() {
    document.getElementById('id01').style.display='none';
    gamenameEl = document.getElementById('gamename');
    let result = {
        Name: gamenameEl.value,
        Windows: [],
        MacOS: [],
        Linux: []
    };

    ["Windows", "MacOS", "Linux"].forEach(element => {
        pathElems = document.getElementsByClassName(`${element}-path`)
        extElems = document.getElementsByClassName(`${element}-ext`)
        ignoreElems = document.getElementsByClassName(`${element}-ignore`)
        downloadElems = document.getElementsByClassName(`${element}-download`)
        uploadElems = document.getElementsByClassName(`${element}-upload`)
        deleteElems = document.getElementsByClassName(`${element}-delete`)

        for (let i = 0; i < pathElems.length; ++i) {
            pathEl = pathElems[i];
            extEl = extElems[i];
            ignoreEl = ignoreElems[i];
            downloadEl = downloadElems[i];
            uploadEl = uploadElems[i];
            deleteEl = deleteElems[i];

            let extensions = [];
            if (extEl.value) {
                extensions = extEl.value.split(',') || [extEl.value];
            }
    
            let ignoreList = [];
            if (ignoreEl.value) {
                ignoreList = ignoreEl.value.split(',') || [ignoreEl.value];
            }
    
            result[element].push({
                Path: pathEl.value || "",
                Exts: extensions,
                Ignore: ignoreList,
                Download: downloadEl.checked,
                Upload: uploadEl.checked,
                Delete: deleteEl.checked
            });
        }
    });

    if (pendingEdit) {
        removeGamedefByKey(pendingEdit);
        pendingEdit = null;
    }

    commitGamedef(result)
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