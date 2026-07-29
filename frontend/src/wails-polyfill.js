// Polyfill & Bridge for running Awd DriveRouter in Web Browsers / Headless Web Server mode
window.isWebMode = !window.wails;

if (!window['go']) {
  window['go'] = {
    main: {
      App: new Proxy({}, {
        get(target, prop) {
          return async (...args) => {
            const methodName = String(prop);

            // Handle Desktop-only Native Dialog methods gracefully in Web mode
            if (methodName.includes('Dialog') || methodName === 'SelectBackupFolder') {
              alert('⚠️ Fitur dialog berkas native OS ini hanya tersedia di Aplikasi Desktop GUI.\n\nUntuk mengunggah berkas di Web Mode, silakan gunakan tombol Unggah Web atau seret berkas (Drag & Drop) ke antarmuka.');
              return null;
            }

            try {
              let apiKey = localStorage.getItem('api_key') || '';
              let res = await fetch('/api/' + methodName, {
                method: 'POST',
                headers: {
                  'Content-Type': 'application/json',
                  'Authorization': 'Bearer ' + apiKey
                },
                body: JSON.stringify(args)
              });

              if (res.status === 401) {
                const userKey = prompt('🔒 Awd DriveRouter Server dilindungi kunci keamanan.\n\nMasukkan API Key / Password server Anda:');
                if (userKey) {
                  localStorage.setItem('api_key', userKey.trim());
                  // Retry request with newly saved API key
                  return await window['go']['main']['App'][prop](...args);
                }
              }

              if (!res.ok) {
                res = await fetch('/api/' + methodName, {
                  headers: {
                    'Authorization': 'Bearer ' + apiKey
                  }
                });
              }
              if (res.ok) {
                const data = await res.json();
                return data;
              }
            } catch (e) {
              console.warn('Wails Web Proxy API call fallback error:', e);
            }
            if (methodName.startsWith('GetSettings')) return {};
            return [];
          };
        }
      })
    }
  };
}

if (!window['runtime']) {
  window['runtime'] = {
    EventsOn: function() { return function() {}; },
    EventsOff: function() { return function() {}; },
    EventsOnMultiple: function() { return function() {}; },
    EventsOnce: function() { return function() {}; },
    EventsEmit: function() {},
    WindowSetTitle: function() {},
    WindowMinimise: function() {},
    WindowMaximise: function() {},
    WindowToggleMaximise: function() {},
    WindowUnmaximise: function() {},
    WindowHide: function() {},
    WindowShow: function() {},
    WindowReload: function() {},
    WindowReloadApp: function() {},
    WindowSetSystemDefaultTheme: function() {},
    WindowSetLightingMode: function() {},
    WindowSetAlwaysOnTop: function() {},
    WindowSetPosition: function() {},
    WindowGetPosition: function() { return Promise.resolve({x:0, y:0}); },
    WindowSetSize: function() {},
    WindowGetSize: function() { return Promise.resolve({w:1024, h:768}); },
    WindowSetMinSize: function() {},
    WindowSetMaxSize: function() {},
    WindowSetBackgroundColour: function() {},
    ScreenGetAll: function() { return Promise.resolve([]); },
    BrowserOpenURL: function(url) { window.open(url, '_blank'); },
    Environment: function() { return Promise.resolve({buildType:'production', platform:'web', arch:'amd64'}); }
  };
}
