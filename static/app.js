(function() {
    'use strict';

    // Elements
    const loadingEl = document.getElementById('loading');
    const errorEl = document.getElementById('error');
    const errorMessageEl = document.getElementById('error-message');
    const displayEl = document.getElementById('display');
    const levelNumberEl = document.getElementById('level-number');
    const descriptorEl = document.getElementById('descriptor');
    const descriptionEl = document.getElementById('description');
    const dewpointEl = document.getElementById('dewpoint');
    const locationEl = document.getElementById('location');
    const stationEl = document.getElementById('station');
    const timestampEl = document.getElementById('timestamp');
    const refreshBtn = document.getElementById('refresh-btn');
    const locationBtn = document.getElementById('location-btn');
    const retryBtn = document.getElementById('retry-btn');
    const modalEl = document.getElementById('location-modal');
    const zipInput = document.getElementById('zip-input');
    const modalCancel = document.getElementById('modal-cancel');
    const modalSubmit = document.getElementById('modal-submit');

    // State
    let currentLat = null;
    let currentLon = null;
    let currentZip = null;

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
                fetchJuicecon();
            },
            function(error) {
                console.log('Geolocation error:', error);
                showModal();
            },
            { timeout: 10000 }
        );
    }

    // API
    function fetchJuicecon() {
        showLoading();

        let url = '/api/juicecon?';
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
        // Set level color
        const level = data.allClear ? 'clear' : data.level;
        document.body.setAttribute('data-level', level);

        // Update level number
        if (data.allClear) {
            levelNumberEl.textContent = 'ALL CLEAR';
            levelNumberEl.classList.add('all-clear');
        } else {
            levelNumberEl.textContent = data.level;
            levelNumberEl.classList.remove('all-clear');
        }

        // Update text
        descriptorEl.textContent = data.descriptor.toUpperCase();
        descriptionEl.textContent = '"' + data.description + '"';

        // Update data panel
        dewpointEl.textContent = data.dewpoint.toFixed(1) + '°F';
        locationEl.textContent = data.location.city + ', ' + data.location.state;
        stationEl.textContent = data.location.station;
        timestampEl.textContent = formatTimestamp(data.timestamp);

        showDisplay();
    }

    function formatTimestamp(isoString) {
        try {
            const date = new Date(isoString);
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
            showError('Please enter a ZIP code to check JUICECON status');
        }
    }

    function submitZip() {
        const zip = zipInput.value.trim();
        if (!/^\d{5}$/.test(zip)) {
            zipInput.style.borderColor = '#ef4444';
            return;
        }

        currentZip = zip;
        currentLat = null;
        currentLon = null;
        hideModal();
        fetchJuicecon();
    }

    function refresh() {
        if (currentZip) {
            fetchJuicecon();
        } else if (currentLat !== null) {
            fetchJuicecon();
        } else {
            getLocation();
        }
    }

    // Start
    init();
})();
