/**
 * Global setup for Playwright E2E tests
 * Handles test environment initialization and cleanup
 */

import { TestDatabase } from "./fixtures/database.js";

class TestSetup {
  constructor() {
    this.testDb = new TestDatabase();
  }

  /**
   * Global setup - runs once before all tests
   */
  async globalSetup() {
    console.log("🔧 Setting up E2E test environment...");

    try {
      // Connect to test database
      await this.testDb.connect();
      console.log("✅ Test database connected");

      // Clean any existing test data
      await this.testDb.cleanTestData();
      console.log("✅ Test data cleaned");

      // Seed basic test data if needed
      await this.testDb.seedTestData();
      console.log("✅ Basic test data seeded");
    } catch (error) {
      console.error("❌ Test setup failed:", error);
      throw error;
    }
  }

  /**
   * Global teardown - runs once after all tests
   */
  async globalTeardown() {
    console.log("🧹 Cleaning up E2E test environment...");

    try {
      if (this.testDb) {
        await this.testDb.disconnect();
        console.log("✅ Test database disconnected");
      }
    } catch (error) {
      console.warn("⚠️ Cleanup warning:", error.message);
    }
  }

  /**
   * Test isolation setup - runs before each test file
   */
  async testFileSetup() {
    // Clean test data between test files
    if (this.testDb.db) {
      await this.testDb.cleanTestData();
    }
  }

  /**
   * Test isolation teardown - runs after each test file
   */
  async testFileTeardown() {
    // Additional cleanup if needed
    if (this.testDb.db) {
      const stats = await this.testDb.getTestStats();
      console.log("📊 Test data stats:", stats);
    }
  }
}

const testSetup = new TestSetup();

// Export default function for Playwright globalSetup
export default async function globalSetup() {
  await testSetup.globalSetup();
}

// Export teardown function separately
export const globalTeardown = async () => {
  await testSetup.globalTeardown();
};
