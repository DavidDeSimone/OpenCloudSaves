let pendingEdit = null;

async function onSyncButtonClicked(element, name) {
    log(`Sync ${name}`)
    await syncGame(name)
    const interval = setInterval(async () => {
        const logEl = document.getElementById(`${name}-log`);
        const progressEl = document.getElementById(`${name}-progress`);

        logEl.style.display = "block";
        progressEl.style.display = "block";

        const logValue = await pollLogs(name);
        log(logValue)
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
        pathEl = document.getElementById(`${element}-path`);
        extEl = document.getElementById(`${element}-ext`);
        ignoreEl = document.getElementById(`${element}-ignore`);
        downloadEl = document.getElementById(`${element}-download`);
        uploadEl = document.getElementById(`${element}-upload`);
        deleteEl = document.getElementById(`${element}-delete`);

        extEl.value = def.Exts ? def.Exts.join(',') : "";
        ignoreEl.value = def.Ignore ? def.Ignore.join(',') : "";
        pathEl.value = def.Path;
        downloadEl.checked = def.Download;
        uploadEl.checked = def.Upload;
        deleteEl.checked = def.Delete;
    });
}

function submitGamedef() {
    document.getElementById('id01').style.display='none';
    gamenameEl = document.getElementById('gamename');
    let result = {
        Name: gamenameEl.value
    };

    ["Windows", "MacOS", "Linux"].forEach(element => {
        pathEl = document.getElementById(`${element}-path`)
        extEl = document.getElementById(`${element}-ext`)
        ignoreEl = document.getElementById(`${element}-ignore`)
        downloadEl = document.getElementById(`${element}-download`)
        uploadEl = document.getElementById(`${element}-upload`)
        deleteEl = document.getElementById(`${element}-delete`)

        let extensions = [];
        if (extEl.value) {
            extensions = extEl.value.split(',') || [extEl.value];
        }

        let ignoreList = [];
        if (ignoreEl.value) {
            ignoreList = ignoreEl.value.split(',') || [ignoreEl.value];
        }

        result[element] = {
            Path: pathEl.value || "",
            Exts: extensions,
            Ignore: ignoreList,
            Download: downloadEl.checked,
            Upload: uploadEl.checked,
            Delete: deleteEl.checked
        };
    });

    if (pendingEdit) {
        removeGamedefByKey(pendingEdit);
        pendingEdit = null;
    }

    commitGamedef(result)
    refresh()
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

function main() {
    setupAccordionHandler();
}

main()