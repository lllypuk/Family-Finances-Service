/**
 * Database utilities for E2E test data management
 * Handles test data seeding, cleanup, and isolation
 */

import { MongoClient } from "mongodb";

export class TestDatabase {
  constructor() {
    this.client = null;
    this.db = null;
    this.testCollections = [
      "users",
      "families",
      "transactions",
      "categories",
      "budgets",
      "reports",
    ];
    this.testDataIds = new Set();
  }

  /**
   * Connect to test database
   */
  async connect() {
    if (this.client) {
      return this.db;
    }

    // Use test-specific database name
    const testDbName = `family_budget_test_${Date.now()}`;
    const mongoUri =
      process.env.MONGODB_URI || "mongodb://admin:password123@localhost:27017";

    this.client = new MongoClient(mongoUri);
    await this.client.connect();
    this.db = this.client.db(testDbName);

    console.log(`Connected to test database: ${testDbName}`);
    return this.db;
  }

  /**
   * Disconnect and cleanup test database
   */
  async disconnect() {
    if (this.client) {
      // Drop the entire test database
      if (this.db) {
        await this.db.dropDatabase();
        console.log(`Test database ${this.db.databaseName} dropped`);
      }

      await this.client.close();
      this.client = null;
      this.db = null;
      this.testDataIds.clear();
    }
  }

  /**
   * Clean all test data from collections
   */
  async cleanTestData() {
    if (!this.db) {
      await this.connect();
    }

    const cleanupPromises = this.testCollections.map(async (collectionName) => {
      const collection = this.db.collection(collectionName);

      // Delete documents that have test_id field or match test patterns
      await collection.deleteMany({
        $or: [
          { test_id: { $exists: true } },
          { email: { $regex: /test.*@example\.com/i } },
          { name: { $regex: /^Test\s/i } },
          { family_name: { $regex: /^Test\s/i } },
        ],
      });
    });

    await Promise.all(cleanupPromises);
    console.log("Test data cleaned from all collections");
  }

  /**
   * Seed basic test data (categories, etc.)
   */
  async seedTestData() {
    if (!this.db) {
      await this.connect();
    }

    // Default test categories
    const defaultCategories = [
      {
        _id: this.generateTestId("category"),
        name: "Test Income",
        type: "income",
        color: "#4CAF50",
        icon: "money",
        family_id: "test-family",
        test_id: "test-category-income",
      },
      {
        _id: this.generateTestId("category"),
        name: "Test Expense",
        type: "expense",
        color: "#F44336",
        icon: "shopping",
        family_id: "test-family",
        test_id: "test-category-expense",
      },
      {
        _id: this.generateTestId("category"),
        name: "Test Food",
        type: "expense",
        color: "#FF9800",
        icon: "food",
        family_id: "test-family",
        parent_id: null,
        test_id: "test-category-food",
      },
    ];

    const categoriesCollection = this.db.collection("categories");
    await categoriesCollection.insertMany(defaultCategories);

    console.log("Test categories seeded");
    return defaultCategories;
  }

  /**
   * Create test user in database
   */
  async createTestUser(userData) {
    if (!this.db) {
      await this.connect();
    }

    const usersCollection = this.db.collection("users");

    const testUser = {
      _id: this.generateTestId("user"),
      ...userData,
      created_at: new Date(),
      updated_at: new Date(),
      is_active: true,
      // Add test identifier
      test_id: userData.test_id || `test-user-${Date.now()}`,
    };

    await usersCollection.insertOne(testUser);
    this.testDataIds.add(testUser._id);

    return testUser;
  }

  /**
   * Create test family in database
   */
  async createTestFamily(familyData) {
    if (!this.db) {
      await this.connect();
    }

    const familiesCollection = this.db.collection("families");

    const testFamily = {
      _id: this.generateTestId("family"),
      ...familyData,
      created_at: new Date(),
      updated_at: new Date(),
      settings: {
        currency: "RUB",
        timezone: "Europe/Moscow",
        ...familyData.settings,
      },
      test_id: familyData.test_id || `test-family-${Date.now()}`,
    };

    await familiesCollection.insertOne(testFamily);
    this.testDataIds.add(testFamily._id);

    return testFamily;
  }

  /**
   * Create test transaction
   */
  async createTestTransaction(transactionData) {
    if (!this.db) {
      await this.connect();
    }

    const transactionsCollection = this.db.collection("transactions");

    const testTransaction = {
      _id: this.generateTestId("transaction"),
      ...transactionData,
      created_at: new Date(),
      updated_at: new Date(),
      test_id: transactionData.test_id || `test-transaction-${Date.now()}`,
    };

    await transactionsCollection.insertOne(testTransaction);
    this.testDataIds.add(testTransaction._id);

    return testTransaction;
  }

  /**
   * Generate test-specific ObjectId
   */
  generateTestId(prefix = "test") {
    // Generate MongoDB-compatible ObjectId for tests
    const timestamp = Math.floor(Date.now() / 1000);
    const randomBytes = Math.random()
      .toString(16)
      .substring(2, 18)
      .padStart(16, "0");
    return `${timestamp.toString(16).padStart(8, "0")}${randomBytes}`;
  }

  /**
   * Get test database statistics
   */
  async getTestStats() {
    if (!this.db) {
      return null;
    }

    const stats = {};

    for (const collectionName of this.testCollections) {
      const collection = this.db.collection(collectionName);
      const count = await collection.countDocuments({
        test_id: { $exists: true },
      });
      stats[collectionName] = count;
    }

    return stats;
  }

  /**
   * Verify data isolation between tests
   */
  async verifyDataIsolation(familyId) {
    if (!this.db) {
      return false;
    }

    // Check that no data leaks between different families
    const collections = ["transactions", "categories", "budgets"];

    for (const collectionName of collections) {
      const collection = this.db.collection(collectionName);
      const crossFamilyData = await collection.findOne({
        family_id: { $ne: familyId, $exists: true },
        test_id: { $exists: true },
      });

      if (crossFamilyData) {
        console.error(
          `Data isolation violation in ${collectionName}:`,
          crossFamilyData,
        );
        return false;
      }
    }

    return true;
  }

  /**
   * Setup test environment for specific test
   */
  async setupTestEnvironment(scenario) {
    await this.connect();
    await this.cleanTestData();

    if (scenario.seedData) {
      await this.seedTestData();
    }

    // Create any required test data based on scenario
    if (scenario.users) {
      for (const userData of scenario.users) {
        await this.createTestUser(userData);
      }
    }

    if (scenario.families) {
      for (const familyData of scenario.families) {
        await this.createTestFamily(familyData);
      }
    }

    console.log(
      `Test environment setup complete for scenario: ${scenario.name}`,
    );
  }
}
