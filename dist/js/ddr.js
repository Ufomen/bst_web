function updateProfile() {
    let xhttp = new XMLHttpRequest();
    let btn = document.getElementById('ddr-player-details-update');
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            if (this.status == 200) {
                processing.innerText = "Done!"
            } else if (this.status == 500) {
                processing.innerText = this.response
            }
        }
    };

    btn.style.display = 'none';

    let processing = document.createElement('span');
    processing.id = 'ddr-update-processing'
    processing.appendChild(document.createTextNode('processing ...'));
    btn.parentNode.insertBefore(processing, btn);

    xhttp.open("PATCH", "/external/bst_api/ddr_update", true);
    xhttp.send();
}

function refreshProfile() {
    let xhttp = new XMLHttpRequest();
    let btn = document.getElementById('ddr-player-details-refresh');
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            if (this.status == 200) {
                processing.innerText = "Done!"
            } else if (this.status == 500) {
                processing.innerText = this.response
            }
        }
    };

    btn.style.display = 'none';

    let processing = document.createElement('span');
    processing.id = 'ddr-refresh-processing'
    processing.appendChild(document.createTextNode('processing ...'));
    btn.parentNode.insertBefore(processing, btn);
    xhttp.open("PATCH", "/external/bst_api/ddr_refresh", true);
    xhttp.send();
}