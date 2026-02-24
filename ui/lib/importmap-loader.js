// Import map loader - loads the import map from external JSON file
(async function() {
  try {
    const response = await fetch('/importmap.json');
    const importMap = await response.json();

    const script = document.createElement('script');
    script.type = 'importmap';
    script.textContent = JSON.stringify(importMap);
    document.head.appendChild(script);
  } catch (error) {
    console.error('Failed to load import map:', error);
  }
})();