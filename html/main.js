function onSyncButtonClicked(element, name) {
    log(`Sync ${name}`)
    syncGame(name)
}

function onEditButtonClicked(element, name) {
    log(`Edit ${name}`)
}

function onRemoveButtonClicked(element, name) {
    log(`Remove ${name}`)
}

function onAddGameClosed() {
    document.getElementById('id01').style.display='none';
    refresh();
}

function deserGamedef(gamedef) {
    ["Windows", "MacOS", "Linux"].forEach(element => {
        const def = gamedef[element];
        pathEl = document.getElementById(`${element}-path`);
        extEl = document.getElementById(`${element}-ext`);
        ignoreEl = document.getElementById(`${element}-ignore`);
        downloadEl = document.getElementById(`${element}-download`);
        uploadEl = document.getElementById(`${element}-upload`);
        deleteEl = document.getElementById(`${element}-delete`);

        extEl.value = def.Exts.join(',');
        ignoreEl.value = def.Ignore.join(',');
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
            extensions = extEl.value.split(',');
        }

        let ignoreList = [];
        if (ignoreEl.value) {
            ignoreList = ignoreList.value.split(',');
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

    const s = JSON.stringify(result)
    log(s)
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