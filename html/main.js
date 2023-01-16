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

function setupModalHandler() {
    // Get the modal
    var modal = document.getElementById('id01');

    // When the user clicks anywhere outside of the modal, close it
    window.onclick = function(event) {
        if (event.target == modal) {
            modal.style.display = "none";
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

function main() {
    setupModalHandler();
    setupAccordionHandler();
}

try { 
    main()
} catch (e) {
    log("" + e)
}