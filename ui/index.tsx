import * as React from 'react';
import {createRoot} from 'react-dom/client';
import App from './App';

// Suppress benign ResizeObserver errors commonly caused by React Flow or layout libraries
// These errors are generally safe to ignore in this context
const resizeObserverLoopErr = /ResizeObserver loop limit exceeded|ResizeObserver loop completed with undelivered notifications/;

const originalOnError = window.onerror;
window.onerror = function (msg, url, lineNo, columnNo, error) {
    if (resizeObserverLoopErr.test(msg as string)) {
        return true; // Stop propagation
    }
    if (originalOnError) {
        return originalOnError(msg, url, lineNo, columnNo, error);
    }
    return false;
};

window.addEventListener('error', (e) => {
    if (resizeObserverLoopErr.test(e.message)) {
        e.stopImmediatePropagation();
    }
});

const container = document.getElementById('root');
if (!container) {
    throw new Error("Could not find root element to mount to");
}

const root = createRoot(container);
root.render(
    <React.StrictMode>
        <App/>
    </React.StrictMode>
);