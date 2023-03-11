function makeConfirmationPopup(args) {
    const modal = document.getElementById('confirmation-modal');
    const title = document.getElementById('confirmation-title');
    const subtitle = document.getElementById('confirmation-subtitle');
    const confirmButton = document.getElementById('confirmation-modal-confirm');
    const cancelButton = document.getElementById('confirmation-modal-cancel');

    title.innerText = args.title || '';
    subtitle.innerText = args.subtitle || '';
    confirmButton.onclick = () => {
        if (args.onConfirm) {
            args.onConfirm();
        }
        modal.style.display = 'none';
    };

    cancelButton.onclick = () => {
        if (args.onCancel) {
            args.onCancel();
        }

        modal.style.display = 'none';
    };


    modal.style.display = 'block';
}