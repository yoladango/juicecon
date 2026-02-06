(function() {
    'use strict';

    // Elements
    var loadingEl = document.getElementById('loading');
    var errorEl = document.getElementById('error');
    var errorMessageEl = document.getElementById('error-message');
    var displayEl = document.getElementById('display');
    var levelNumberEl = document.getElementById('level-number');
    var levelPrefixEl = document.getElementById('level-prefix');
    var descriptorEl = document.getElementById('descriptor');
    var descriptionEl = document.getElementById('description');
    var protocolEl = document.getElementById('protocol');
    var temperatureEl = document.getElementById('temperature');
    var dewpointEl = document.getElementById('dewpoint');
    var locationEl = document.getElementById('location');
    var stationEl = document.getElementById('station');
    var timestampEl = document.getElementById('timestamp');
    var refreshBtn = document.getElementById('refresh-btn');
    var locationBtn = document.getElementById('location-btn');
    var retryBtn = document.getElementById('retry-btn');
    var modalEl = document.getElementById('location-modal');
    var zipInput = document.getElementById('zip-input');
    var modalCancel = document.getElementById('modal-cancel');
    var modalSubmit = document.getElementById('modal-submit');
    var infoBtn = document.getElementById('info-btn');
    var infoModal = document.getElementById('info-modal');
    var infoClose = document.getElementById('info-close');

    // State
    var currentLat = null;
    var currentLon = null;
    var currentZip = null;

    // System display names
    var systemNames = {
        juicecon: 'JUICECON',
        ccf: 'CUNNINGHAM CRACKLE FACTOR',
        none: 'DEWCON'
    };

    // Initialize
    function init() {
        bindEvents();
        getLocation();
    }

    function bindEvents() {
        refreshBtn.addEventListener('click', refresh);
        locationBtn.addEventListener('click', showModal);
        retryBtn.addEventListener('click', refresh);
        modalCancel.addEventListener('click', hideModal);
        modalSubmit.addEventListener('click', submitZip);
        zipInput.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') submitZip();
        });
        infoBtn.addEventListener('click', function() {
            infoModal.classList.remove('hidden');
        });
        infoClose.addEventListener('click', function() {
            infoModal.classList.add('hidden');
        });
    }

    // Location
    function getLocation() {
        showLoading();

        if (!navigator.geolocation) {
            showModal();
            return;
        }

        navigator.geolocation.getCurrentPosition(
            function(position) {
                currentLat = position.coords.latitude;
                currentLon = position.coords.longitude;
                currentZip = null;
                fetchData();
            },
            function(error) {
                console.log('Geolocation error:', error);
                showModal();
            },
            { timeout: 10000 }
        );
    }

    // API
    function fetchData() {
        showLoading();

        var url = '/api/juicecon?';
        if (currentZip) {
            url += 'zip=' + encodeURIComponent(currentZip);
        } else if (currentLat !== null && currentLon !== null) {
            url += 'lat=' + currentLat + '&lon=' + currentLon;
        } else {
            showError('No location available');
            return;
        }

        fetch(url)
            .then(function(response) {
                return response.json().then(function(data) {
                    if (!response.ok) {
                        throw new Error(data.error || 'Unknown error');
                    }
                    return data;
                });
            })
            .then(function(data) {
                updateDisplay(data);
            })
            .catch(function(error) {
                showError(error.message);
            });
    }

    // Display
    function updateDisplay(data) {
        var system = data.activeSystem || 'none';
        var level = data.allClear ? 'clear' : data.level;

        // Set system and level on body for CSS color switching
        document.body.setAttribute('data-system', system);
        document.body.setAttribute('data-level', level);

        // Update level number
        if (data.allClear) {
            levelNumberEl.textContent = 'ALL CLEAR';
            levelNumberEl.classList.add('all-clear');
        } else {
            levelNumberEl.textContent = data.level;
            levelNumberEl.classList.remove('all-clear');
        }

        // Update system prefix label
        if (data.allClear && system === 'none') {
            levelPrefixEl.textContent = 'DEWCON';
        } else {
            levelPrefixEl.textContent = data.systemName || 'DEWCON';
        }

        // Update text
        descriptorEl.textContent = data.descriptor.toUpperCase();
        descriptionEl.textContent = '"' + data.description + '"';

        // Update data panel
        protocolEl.textContent = systemNames[system] || 'DEWCON';
        temperatureEl.textContent = data.temperature ? data.temperature.toFixed(1) + '\u00B0F' : '--\u00B0F';
        dewpointEl.textContent = data.dewpoint.toFixed(1) + '\u00B0F';
        locationEl.textContent = data.location.city + ', ' + data.location.state;
        stationEl.textContent = data.location.station;
        timestampEl.textContent = formatTimestamp(data.timestamp);

        showDisplay();
    }

    function formatTimestamp(isoString) {
        try {
            var date = new Date(isoString);
            return date.toLocaleTimeString('en-US', {
                hour: 'numeric',
                minute: '2-digit',
                hour12: true,
                timeZoneName: 'short'
            });
        } catch (e) {
            return '--:-- --';
        }
    }

    // UI State
    function showLoading() {
        loadingEl.classList.remove('hidden');
        errorEl.classList.add('hidden');
        displayEl.classList.add('hidden');
    }

    function showError(message) {
        loadingEl.classList.add('hidden');
        errorEl.classList.remove('hidden');
        displayEl.classList.add('hidden');
        errorMessageEl.textContent = message;
    }

    function showDisplay() {
        loadingEl.classList.add('hidden');
        errorEl.classList.add('hidden');
        displayEl.classList.remove('hidden');
    }

    // Modal
    function showModal() {
        loadingEl.classList.add('hidden');
        modalEl.classList.remove('hidden');
        zipInput.value = '';
        zipInput.focus();
    }

    function hideModal() {
        modalEl.classList.add('hidden');
        // If we don't have a location, show error
        if (currentLat === null && currentZip === null) {
            showError('Please enter a ZIP code to check DEWCON status');
        }
    }

    function submitZip() {
        var zip = zipInput.value.trim();
        if (!/^\d{5}$/.test(zip)) {
            zipInput.style.borderColor = '#ef4444';
            return;
        }

        currentZip = zip;
        currentLat = null;
        currentLon = null;
        hideModal();
        fetchData();
    }

    function refresh() {
        if (currentZip) {
            fetchData();
        } else if (currentLat !== null) {
            fetchData();
        } else {
            getLocation();
        }
    }

    // Start
    init();
})();
