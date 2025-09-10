/**
 * User test data fixtures for E2E tests
 * Provides consistent test user data with role-based access
 */

export class UserFactory {
  constructor() {
    this.testUsers = new Map();
  }

  /**
   * Generate test user data with unique identifiers
   */
  createUser(role = "admin", options = {}) {
    const timestamp = Date.now();
    const uniqueId = Math.random().toString(36).substring(7);

    const baseUser = {
      name: options.name || `Test User ${uniqueId}`,
      family_name: options.family_name || `Test Family ${uniqueId}`,
      currency: options.currency || "RUB",
      email: options.email || `test-${role}-${timestamp}@example.com`,
      password: options.password || "TestPassword123!",
      confirm_password: options.confirm_password || "TestPassword123!",
      role: role,
      created_at: new Date().toISOString(),
      test_id: `${role}-${uniqueId}-${timestamp}`,
    };

    // Store for cleanup
    this.testUsers.set(baseUser.test_id, baseUser);

    return baseUser;
  }

  /**
   * Create family admin user
   */
  createFamilyAdmin(options = {}) {
    return this.createUser("admin", {
      name: "Family Administrator",
      family_name: options.family_name || `Admin Family ${Date.now()}`,
      ...options,
    });
  }

  /**
   * Create family member user
   */
  createFamilyMember(familyName, options = {}) {
    return this.createUser("member", {
      name: "Family Member",
      family_name: familyName,
      ...options,
    });
  }

  /**
   * Create child user
   */
  createChild(familyName, options = {}) {
    return this.createUser("child", {
      name: "Family Child",
      family_name: familyName,
      ...options,
    });
  }

  /**
   * Generate test family with multiple users
   */
  createTestFamily(options = {}) {
    const familyName = options.family_name || `Test Family ${Date.now()}`;

    const family = {
      name: familyName,
      admin: this.createFamilyAdmin({ family_name: familyName }),
      members: [],
      children: [],
    };

    // Add members if requested
    const memberCount = options.memberCount || 1;
    for (let i = 0; i < memberCount; i++) {
      family.members.push(
        this.createFamilyMember(familyName, {
          name: `Family Member ${i + 1}`,
        }),
      );
    }

    // Add children if requested
    const childCount = options.childCount || 0;
    for (let i = 0; i < childCount; i++) {
      family.children.push(
        this.createChild(familyName, {
          name: `Family Child ${i + 1}`,
        }),
      );
    }

    return family;
  }

  /**
   * Get all test users created
   */
  getAllTestUsers() {
    return Array.from(this.testUsers.values());
  }

  /**
   * Clear all test users
   */
  clearUsers() {
    this.testUsers.clear();
  }

  /**
   * Get test user by ID
   */
  getUser(testId) {
    return this.testUsers.get(testId);
  }
}

// Pre-defined test scenarios
export const TEST_SCENARIOS = {
  SINGLE_USER: {
    description: "Single admin user scenario",
    setup: (factory) => ({
      admin: factory.createFamilyAdmin(),
    }),
  },

  SMALL_FAMILY: {
    description: "Small family with admin and member",
    setup: (factory) =>
      factory.createTestFamily({
        memberCount: 1,
        childCount: 0,
      }),
  },

  LARGE_FAMILY: {
    description: "Large family with multiple users",
    setup: (factory) =>
      factory.createTestFamily({
        memberCount: 3,
        childCount: 2,
      }),
  },

  MULTI_FAMILY: {
    description: "Multiple families for isolation testing",
    setup: (factory) => ({
      family1: factory.createTestFamily({
        family_name: `Family A ${Date.now()}`,
        memberCount: 2,
      }),
      family2: factory.createTestFamily({
        family_name: `Family B ${Date.now()}`,
        memberCount: 1,
      }),
    }),
  },
};

// Invalid user data for validation testing
export const INVALID_USER_DATA = {
  MISSING_FIELDS: {
    name: "",
    family_name: "",
    email: "",
    password: "",
  },

  INVALID_EMAIL: {
    name: "Test User",
    family_name: "Test Family",
    email: "invalid-email",
    password: "TestPassword123!",
  },

  WEAK_PASSWORD: {
    name: "Test User",
    family_name: "Test Family",
    email: "test@example.com",
    password: "123",
  },

  PASSWORD_MISMATCH: {
    name: "Test User",
    family_name: "Test Family",
    email: "test@example.com",
    password: "TestPassword123!",
    confirm_password: "DifferentPassword123!",
  },

  LONG_FIELDS: {
    name: "A".repeat(256),
    family_name: "B".repeat(256),
    email: "test@example.com",
    password: "TestPassword123!",
  },
};
