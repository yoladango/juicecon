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
    var refreshStatusEl = document.getElementById('refresh-status');
    var refreshStatusTextEl = document.getElementById('refresh-status-text');

    // State
    var currentLat = null;
    var currentLon = null;
    var currentZip = null;
    var autoRefreshTimer = null;
    var lastRefreshTime = null;
    var modalTriggerElement = null;

    var AUTO_REFRESH_INTERVAL = 15 * 60 * 1000; // 15 minutes

    // System display names
    var systemNames = {
        juicecon: 'JUICECON',
        ccf: 'CUNNINGHAM CRACKLE FACTOR',
        none: 'DEWCON'
    };

    // Check for test mode dewpoint override via URL param
    var urlParams = new URLSearchParams(window.location.search);
    var testDewpoint = urlParams.get('_dewpoint');

    // Initialize
    function init() {
        bindEvents();
        if (testDewpoint !== null) {
            fetchTestDewpoint(testDewpoint);
        } else {
            getLocation();
        }
    }

    function bindEvents() {
        refreshBtn.addEventListener('click', manualRefresh);
        locationBtn.addEventListener('click', function() {
            openModal(modalEl, locationBtn);
        });
        retryBtn.addEventListener('click', manualRefresh);
        modalCancel.addEventListener('click', function() {
            closeModal(modalEl);
        });
        modalSubmit.addEventListener('click', submitZip);
        zipInput.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') submitZip();
        });
        infoBtn.addEventListener('click', function() {
            openModal(infoModal, infoBtn);
        });
        infoClose.addEventListener('click', function() {
            closeModal(infoModal);
        });

        // Global Escape key handler for modals
        document.addEventListener('keydown', function(e) {
            if (e.key === 'Escape') {
                if (!infoModal.classList.contains('hidden')) {
                    closeModal(infoModal);
                } else if (!modalEl.classList.contains('hidden')) {
                    closeModal(modalEl);
                }
            }
        });
    }

    // =====================
    // Modal focus management
    // =====================

    function getFocusableElements(container) {
        var elements = container.querySelectorAll(
            'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        );
        return Array.prototype.filter.call(elements, function(el) {
            return !el.disabled && el.offsetParent !== null;
        });
    }

    function openModal(modal, triggerEl) {
        modalTriggerElement = triggerEl || document.activeElement;
        modal.classList.remove('hidden');

        // If this is the location modal, also hide loading
        if (modal === modalEl) {
            loadingEl.classList.add('hidden');
            zipInput.value = '';
        }

        // Focus first focusable element
        var focusable = getFocusableElements(modal);
        if (modal === modalEl && zipInput) {
            zipInput.focus();
        } else if (focusable.length > 0) {
            focusable[0].focus();
        }

        // Attach focus trap
        modal._trapHandler = function(e) {
            trapFocus(e, modal);
        };
        modal.addEventListener('keydown', modal._trapHandler);
    }

    function closeModal(modal) {
        modal.classList.add('hidden');

        // Remove focus trap
        if (modal._trapHandler) {
            modal.removeEventListener('keydown', modal._trapHandler);
            modal._trapHandler = null;
        }

        // Restore focus to trigger element
        if (modalTriggerElement && typeof modalTriggerElement.focus === 'function') {
            modalTriggerElement.focus();
            modalTriggerElement = null;
        }

        // If location modal closed with no location, show error
        if (modal === modalEl && currentLat === null && currentZip === null) {
            showError('Please enter a ZIP code to check DEWCON status');
        }
    }

    function trapFocus(e, modal) {
        if (e.key !== 'Tab') return;

        var focusable = getFocusableElements(modal);
        if (focusable.length === 0) return;

        var firstEl = focusable[0];
        var lastEl = focusable[focusable.length - 1];

        if (e.shiftKey) {
            // Shift+Tab: if on first element, wrap to last
            if (document.activeElement === firstEl) {
                e.preventDefault();
                lastEl.focus();
            }
        } else {
            // Tab: if on last element, wrap to first
            if (document.activeElement === lastEl) {
                e.preventDefault();
                firstEl.focus();
            }
        }
    }

    // =====================
    // Auto-refresh
    // =====================

    function startAutoRefresh() {
        stopAutoRefresh();
        autoRefreshTimer = setInterval(function() {
            if (currentZip || currentLat !== null) {
                fetchData();
            }
        }, AUTO_REFRESH_INTERVAL);
        updateRefreshStatus();
    }

    function stopAutoRefresh() {
        if (autoRefreshTimer) {
            clearInterval(autoRefreshTimer);
            autoRefreshTimer = null;
        }
    }

    function updateRefreshStatus() {
        if (!lastRefreshTime || !refreshStatusEl) return;

        var timeStr = lastRefreshTime.toLocaleTimeString('en-US', {
            hour: 'numeric',
            minute: '2-digit',
            hour12: true
        });
        refreshStatusTextEl.textContent = 'LAST UPDATED ' + timeStr + ' // NEXT IN 15 MIN';
    }

    function manualRefresh() {
        // Reset the auto-refresh timer on manual refresh
        if (currentZip || currentLat !== null) {
            refresh();
        } else {
            getLocation();
        }
    }

    // Location
    function getLocation() {
        showLoading();

        if (!navigator.geolocation) {
            openModal(modalEl, null);
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
                openModal(modalEl, null);
            },
            { timeout: 10000 }
        );
    }

    // Test mode API call -- bypasses geolocation, sends dewpoint directly
    function fetchTestDewpoint(dp) {
        showLoading();
        fetch('/api/juicecon?dewpoint=' + encodeURIComponent(dp))
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
        temperatureEl.textContent = data.temperature != null ? data.temperature.toFixed(1) + '\u00B0F' : 'N/A';
        dewpointEl.textContent = data.dewpoint.toFixed(1) + '\u00B0F';
        locationEl.textContent = data.location.city + ', ' + data.location.state;
        stationEl.textContent = data.location.station;
        timestampEl.textContent = formatTimestamp(data.timestamp);

        // Track refresh time and start auto-refresh
        lastRefreshTime = new Date();
        startAutoRefresh();

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

    // Modal (legacy wrappers kept for submitZip flow)
    function showModal() {
        openModal(modalEl, null);
    }

    function hideModal() {
        closeModal(modalEl);
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
        closeModal(modalEl);
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
