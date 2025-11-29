/**
 * Integration test configuration
 */

export const INTEGRATION_TEST_CONFIG = {
    /**
     * Backend API base URL
     */
    apiBaseUrl: process.env.VITE_API_BASE_URL || 'http://localhost:8181',

    /**
     * Default timeout for API requests (ms)
     */
    apiTimeout: 30000,

    /**
     * Maximum time to wait for execution completion (ms)
     */
    executionTimeout: 60000,

    /**
     * Polling interval for execution status (ms)
     */
    executionPollInterval: 500,

    /**
     * Maximum retries for flaky operations
     */
    maxRetries: 3,

    /**
     * Initial delay for retry backoff (ms)
     */
    retryInitialDelay: 1000,

    /**
     * Whether to skip integration tests if backend is not available
     */
    skipIfBackendUnavailable: true,

    /**
     * Whether to cleanup test data after tests
     */
    cleanupAfterTests: true,

    /**
     * Test workflow prefix
     */
    testWorkflowPrefix: 'integration-test',

    /**
     * Verbose logging
     */
    verbose: process.env.VITEST_VERBOSE === 'true',
}

/**
 * Test environment checks
 */
export function getTestEnvironment() {
    return {
        isCI: process.env.CI === 'true',
        nodeEnv: process.env.NODE_ENV,
        apiBaseUrl: INTEGRATION_TEST_CONFIG.apiBaseUrl,
    }
}

/**
 * Log test configuration
 */
export function logTestConfig() {
    if (INTEGRATION_TEST_CONFIG.verbose) {
        console.log('Integration Test Configuration:', {
            ...INTEGRATION_TEST_CONFIG,
            environment: getTestEnvironment(),
        })
    }
}
