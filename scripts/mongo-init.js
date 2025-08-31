// MongoDB Initialization Script for Family Budget Service
// This script sets up the database, creates collections, indexes, and default data
print("🚀 Starting MongoDB initialization for Family Budget Service...");

// Switch to the family_budget database
db = db.getSiblingDB("family_budget");

print("🗂️  Creating collections with updated validation schemas...");

// Users collection
db.createCollection("users", {
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: [
        "_id",
        "email",
        "password",
        "first_name",
        "last_name",
        "role",
        "family_id",
        "created_at",
        "updated_at",
      ],
      properties: {
        _id: { bsonType: "binData" }, // UUID as binary data
        email: {
          bsonType: "string",
          pattern: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
          description: "Valid email address",
        },
        password: {
          bsonType: "string",
          minLength: 1,
          description: "Bcrypt hashed password",
        },
        first_name: {
          bsonType: "string",
          minLength: 1,
          maxLength: 100,
        },
        last_name: {
          bsonType: "string",
          minLength: 1,
          maxLength: 100,
        },
        role: {
          bsonType: "string",
          enum: ["admin", "member", "child"],
          description: "User role within family",
        },
        family_id: { bsonType: "binData" },
        created_at: { bsonType: "date" },
        updated_at: { bsonType: "date" },
      },
    },
  },
});

// Families collection
db.createCollection("families", {
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: ["_id", "name", "currency", "created_at", "updated_at"],
      properties: {
        _id: { bsonType: "binData" },
        name: {
          bsonType: "string",
          minLength: 1,
          maxLength: 200,
        },
        currency: {
          bsonType: "string",
          pattern: "^[A-Z]{3}$",
          description: "ISO 4217 currency code (USD, RUB, EUR, etc.)",
        },
        created_at: { bsonType: "date" },
        updated_at: { bsonType: "date" },
      },
    },
  },
});

// Categories collection
db.createCollection("categories", {
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: [
        "_id",
        "name",
        "type",
        "family_id",
        "is_active",
        "created_at",
        "updated_at",
      ],
      properties: {
        _id: { bsonType: "binData" },
        name: {
          bsonType: "string",
          minLength: 1,
          maxLength: 100,
        },
        type: {
          bsonType: "string",
          enum: ["income", "expense"],
          description: "Category type",
        },
        color: {
          bsonType: "string",
          pattern: "^#[0-9A-Fa-f]{6}$",
          description: "Hex color code for UI",
        },
        icon: {
          bsonType: "string",
          description: "Icon name or emoji for UI",
        },
        parent_id: {
          bsonType: ["binData", "null"],
          description: "Parent category ID for subcategories",
        },
        family_id: { bsonType: "binData" },
        is_active: { bsonType: "bool" },
        created_at: { bsonType: "date" },
        updated_at: { bsonType: "date" },
      },
    },
  },
});

// Transactions collection
db.createCollection("transactions", {
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: [
        "_id",
        "amount",
        "type",
        "category_id",
        "user_id",
        "family_id",
        "date",
        "created_at",
        "updated_at",
      ],
      properties: {
        _id: { bsonType: "binData" },
        amount: {
          bsonType: "double",
          minimum: 0,
          description: "Transaction amount (positive)",
        },
        type: {
          bsonType: "string",
          enum: ["income", "expense"],
          description: "Transaction type",
        },
        description: {
          bsonType: "string",
          maxLength: 500,
          description: "Transaction description",
        },
        category_id: { bsonType: "binData" },
        user_id: {
          bsonType: "binData",
          description: "User who created the transaction",
        },
        family_id: { bsonType: "binData" },
        date: {
          bsonType: "date",
          description: "Transaction date",
        },
        tags: {
          bsonType: "array",
          items: {
            bsonType: "string",
            maxLength: 50,
          },
          uniqueItems: true,
          description: "Tags for search and categorization",
        },
        created_at: { bsonType: "date" },
        updated_at: { bsonType: "date" },
      },
    },
  },
});

// Budgets collection
db.createCollection("budgets", {
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: [
        "_id",
        "name",
        "amount",
        "spent",
        "period",
        "family_id",
        "start_date",
        "end_date",
        "is_active",
        "created_at",
        "updated_at",
      ],
      properties: {
        _id: { bsonType: "binData" },
        name: {
          bsonType: "string",
          minLength: 1,
          maxLength: 200,
        },
        amount: {
          bsonType: "double",
          minimum: 0,
          description: "Budget limit amount",
        },
        spent: {
          bsonType: "double",
          minimum: 0,
          description: "Amount already spent",
        },
        period: {
          bsonType: "string",
          enum: ["weekly", "monthly", "yearly", "custom"],
          description: "Budget period",
        },
        category_id: {
          bsonType: ["binData", "null"],
          description: "Optional category restriction",
        },
        family_id: { bsonType: "binData" },
        start_date: { bsonType: "date" },
        end_date: { bsonType: "date" },
        is_active: { bsonType: "bool" },
        created_at: { bsonType: "date" },
        updated_at: { bsonType: "date" },
      },
    },
  },
});

// Reports collection
db.createCollection("reports", {
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: [
        "_id",
        "name",
        "type",
        "period",
        "family_id",
        "user_id",
        "start_date",
        "end_date",
        "data",
        "generated_at",
      ],
      properties: {
        _id: { bsonType: "binData" },
        name: {
          bsonType: "string",
          minLength: 1,
          maxLength: 200,
        },
        type: {
          bsonType: "string",
          enum: ["expenses", "income", "budget", "cash_flow", "category_break"],
          description: "Report type",
        },
        period: {
          bsonType: "string",
          enum: ["daily", "weekly", "monthly", "yearly", "custom"],
          description: "Report period",
        },
        family_id: { bsonType: "binData" },
        user_id: {
          bsonType: "binData",
          description: "User who generated the report",
        },
        start_date: { bsonType: "date" },
        end_date: { bsonType: "date" },
        data: {
          bsonType: "object",
          description: "Report data structure",
        },
        generated_at: { bsonType: "date" },
      },
    },
  },
});

// Sessions collection for web session management
db.createCollection("sessions");

print("📊 Creating indexes for optimal performance...");

// Users indexes
db.users.createIndex(
  { email: 1 },
  { unique: true, name: "users_email_unique" },
);
db.users.createIndex({ family_id: 1 }, { name: "users_family_id" });
db.users.createIndex({ family_id: 1, role: 1 }, { name: "users_family_role" });
db.users.createIndex({ created_at: -1 }, { name: "users_created_at" });

// Families indexes
db.families.createIndex({ name: 1 }, { name: "families_name" });
db.families.createIndex({ created_at: -1 }, { name: "families_created_at" });

// Categories indexes
db.categories.createIndex({ family_id: 1 }, { name: "categories_family_id" });
db.categories.createIndex(
  { family_id: 1, type: 1 },
  { name: "categories_family_type" },
);
db.categories.createIndex(
  { family_id: 1, is_active: 1 },
  { name: "categories_family_active" },
);
db.categories.createIndex({ parent_id: 1 }, { name: "categories_parent_id" });
db.categories.createIndex(
  { family_id: 1, type: 1, is_active: 1 },
  { name: "categories_family_type_active" },
);

// Transactions indexes - optimized for common queries
db.transactions.createIndex(
  { family_id: 1 },
  { name: "transactions_family_id" },
);
db.transactions.createIndex(
  { family_id: 1, date: -1 },
  { name: "transactions_family_date" },
);
db.transactions.createIndex(
  { family_id: 1, type: 1 },
  { name: "transactions_family_type" },
);
db.transactions.createIndex(
  { family_id: 1, category_id: 1 },
  { name: "transactions_family_category" },
);
db.transactions.createIndex(
  { family_id: 1, user_id: 1 },
  { name: "transactions_family_user" },
);
db.transactions.createIndex({ date: -1 }, { name: "transactions_date" });
db.transactions.createIndex({ tags: 1 }, { name: "transactions_tags" });

// Compound indexes for complex queries
db.transactions.createIndex(
  {
    family_id: 1,
    date: -1,
    type: 1,
  },
  { name: "transactions_family_date_type" },
);

db.transactions.createIndex(
  {
    family_id: 1,
    category_id: 1,
    date: -1,
  },
  { name: "transactions_family_category_date" },
);

db.transactions.createIndex(
  {
    family_id: 1,
    type: 1,
    date: -1,
  },
  { name: "transactions_family_type_date" },
);

// Budgets indexes
db.budgets.createIndex({ family_id: 1 }, { name: "budgets_family_id" });
db.budgets.createIndex(
  { family_id: 1, is_active: 1 },
  { name: "budgets_family_active" },
);
db.budgets.createIndex(
  { family_id: 1, period: 1 },
  { name: "budgets_family_period" },
);
db.budgets.createIndex({ category_id: 1 }, { name: "budgets_category_id" });
db.budgets.createIndex(
  { start_date: 1, end_date: 1 },
  { name: "budgets_date_range" },
);
db.budgets.createIndex(
  { family_id: 1, is_active: 1, start_date: 1, end_date: 1 },
  { name: "budgets_family_active_dates" },
);

// Reports indexes
db.reports.createIndex({ family_id: 1 }, { name: "reports_family_id" });
db.reports.createIndex(
  { family_id: 1, type: 1 },
  { name: "reports_family_type" },
);
db.reports.createIndex(
  { family_id: 1, generated_at: -1 },
  { name: "reports_family_generated" },
);
db.reports.createIndex({ user_id: 1 }, { name: "reports_user_id" });

// Sessions indexes with TTL for automatic cleanup
db.sessions.createIndex(
  {
    expires_at: 1,
  },
  {
    expireAfterSeconds: 0,
    name: "sessions_ttl",
  },
);

db.sessions.createIndex(
  {
    session_id: 1,
  },
  {
    unique: true,
    name: "sessions_id_unique",
  },
);

// Text search indexes for better search functionality
db.transactions.createIndex(
  {
    description: "text",
    tags: "text",
  },
  {
    name: "transactions_search",
    default_language: "russian",
    weights: {
      description: 10,
      tags: 5,
    },
  },
);

db.categories.createIndex(
  {
    name: "text",
  },
  {
    name: "categories_search",
    default_language: "russian",
  },
);

print("🏷️  Creating default categories...");

// Updated default expense categories with better icons and colors
const defaultExpenseCategories = [
  { name: "Продукты", color: "#4CAF50", icon: "🛒" },
  { name: "Транспорт", color: "#2196F3", icon: "🚗" },
  { name: "Жилье и ЖКХ", color: "#FF9800", icon: "🏠" },
  { name: "Здоровье и медицина", color: "#F44336", icon: "💊" },
  { name: "Образование", color: "#3F51B5", icon: "📚" },
  { name: "Развлечения", color: "#9C27B0", icon: "🎬" },
  { name: "Одежда и обувь", color: "#E91E63", icon: "👕" },
  { name: "Ресторан и кафе", color: "#FF5722", icon: "🍽️" },
  { name: "Спорт и фитнес", color: "#00BCD4", icon: "⚽" },
  { name: "Хобби и увлечения", color: "#8BC34A", icon: "🎨" },
  { name: "Подарки", color: "#FFC107", icon: "🎁" },
  { name: "Путешествия", color: "#795548", icon: "✈️" },
  { name: "Домашние животные", color: "#607D8B", icon: "🐕" },
  { name: "Красота и уход", color: "#E1BEE7", icon: "💄" },
  { name: "Техника и гаджеты", color: "#37474F", icon: "📱" },
  { name: "Страхование", color: "#546E7A", icon: "🛡️" },
  { name: "Налоги и сборы", color: "#78909C", icon: "📋" },
  { name: "Благотворительность", color: "#A5D6A7", icon: "❤️" },
  { name: "Другое", color: "#9E9E9E", icon: "📦" },
];

// Updated default income categories
const defaultIncomeCategories = [
  { name: "Зарплата", color: "#4CAF50", icon: "💼" },
  { name: "Фриланс", color: "#2196F3", icon: "💻" },
  { name: "Бизнес доходы", color: "#FF9800", icon: "🏢" },
  { name: "Инвестиции", color: "#9C27B0", icon: "📈" },
  { name: "Дивиденды", color: "#3F51B5", icon: "💰" },
  { name: "Подарки", color: "#E91E63", icon: "🎁" },
  { name: "Продажи", color: "#00BCD4", icon: "💸" },
  { name: "Возврат средств", color: "#8BC34A", icon: "↩️" },
  { name: "Аренда", color: "#FF5722", icon: "🏠" },
  { name: "Пособия", color: "#607D8B", icon: "🏛️" },
  { name: "Другое", color: "#9E9E9E", icon: "📦" },
];

// Helper function to generate MongoDB UUID (Binary UUID)
function generateUUID() {
  // Generate a proper UUID4 binary format for MongoDB
  const uuid = UUID();
  return uuid;
}

// Create demo family for initial setup
const currentDate = new Date();
const demoFamilyId = generateUUID();

// Insert demo family
db.families.insertOne({
  _id: demoFamilyId,
  name: "Демо семья",
  currency: "RUB",
  created_at: currentDate,
  updated_at: currentDate,
});

// Insert default expense categories
defaultExpenseCategories.forEach((category) => {
  db.categories.insertOne({
    _id: generateUUID(),
    name: category.name,
    type: "expense",
    color: category.color,
    icon: category.icon,
    family_id: demoFamilyId,
    is_active: true,
    created_at: currentDate,
    updated_at: currentDate,
  });
});

// Insert default income categories
defaultIncomeCategories.forEach((category) => {
  db.categories.insertOne({
    _id: generateUUID(),
    name: category.name,
    type: "income",
    color: category.color,
    icon: category.icon,
    family_id: demoFamilyId,
    is_active: true,
    created_at: currentDate,
    updated_at: currentDate,
  });
});

print("🔍 Creating additional performance indexes...");

// Additional indexes for reporting and analytics
db.transactions.createIndex(
  {
    family_id: 1,
    date: 1,
    amount: 1,
  },
  { name: "transactions_family_date_amount" },
);

db.budgets.createIndex(
  {
    family_id: 1,
    category_id: 1,
    is_active: 1,
  },
  { name: "budgets_family_category_active" },
);

// Partial indexes for active records only (more efficient)
db.categories.createIndex(
  {
    family_id: 1,
    type: 1,
  },
  {
    partialFilterExpression: { is_active: true },
    name: "categories_active_family_type",
  },
);

db.budgets.createIndex(
  {
    family_id: 1,
    start_date: 1,
    end_date: 1,
  },
  {
    partialFilterExpression: { is_active: true },
    name: "budgets_active_family_dates",
  },
);

print("📈 Setting up database optimization...");

// Set up proper read/write concerns for better performance
db.runCommand({
  setDefaultRWConcern: {
    defaultReadConcern: { level: "local" },
    defaultWriteConcern: { w: 1, j: true },
  },
});

print("✅ MongoDB initialization completed successfully!");
print("📋 Summary:");
print("   - 6 collections created with comprehensive validation schemas");
print("   - 35+ optimized indexes for performance");
print("   - Default categories for expenses and income");
print("   - Demo family data for initial setup");
print("   - Text search capabilities for transactions and categories");
print("   - Session management with TTL indexes");
print("   - Performance optimizations enabled");
print("");
print("🚀 Database is ready for Family Budget Service!");
