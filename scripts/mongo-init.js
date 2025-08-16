// MongoDB Initialization Script for Family Budget Service
// This script sets up the database, creates collections, indexes, and default data
print('🚀 Starting MongoDB initialization for Family Budget Service...');
// Switch to the family_budget database
db = db.getSiblingDB('family_budget');
// Create application user with read/write permissions
print('📝 Creating application user...');
db.createUser({
  user: 'family_budget_user',
  pwd: 'family_budget_password',
  roles: [
    {
      role: 'readWrite',
      db: 'family_budget'
    }
  ]
});
// Create collections with validation schemas
print('🗂️  Creating collections...');
// Users collection
db.createCollection('users', {
  validator: {
    $jsonSchema: {
      bsonType: 'object',
      required: ['_id', 'email', 'password', 'first_name', 'last_name', 'role', 'family_id', 'created_at', 'updated_at'],
      properties: {
        _id: { bsonType: 'string' },
        email: { bsonType: 'string', pattern: '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$' },
        password: { bsonType: 'string' },
        first_name: { bsonType: 'string' },
        last_name: { bsonType: 'string' },
        role: { bsonType: 'string', enum: ['admin', 'member', 'child'] },
        family_id: { bsonType: 'string' },
        created_at: { bsonType: 'date' },
        updated_at: { bsonType: 'date' }
      }
    }
  }
});
// Families collection
db.createCollection('families', {
  validator: {
    $jsonSchema: {
      bsonType: 'object',
      required: ['_id', 'name', 'currency', 'created_at', 'updated_at'],
      properties: {
        _id: { bsonType: 'string' },
        name: { bsonType: 'string' },
        currency: { bsonType: 'string' },
        created_at: { bsonType: 'date' },
        updated_at: { bsonType: 'date' }
      }
    }
  }
});
// Categories collection
db.createCollection('categories', {
  validator: {
    $jsonSchema: {
      bsonType: 'object',
      required: ['_id', 'name', 'type', 'family_id', 'is_active', 'created_at', 'updated_at'],
      properties: {
        _id: { bsonType: 'string' },
        name: { bsonType: 'string' },
        type: { bsonType: 'string', enum: ['income', 'expense'] },
        color: { bsonType: 'string' },
        icon: { bsonType: 'string' },
        parent_id: { bsonType: 'string' },
        family_id: { bsonType: 'string' },
        is_active: { bsonType: 'bool' },
        created_at: { bsonType: 'date' },
        updated_at: { bsonType: 'date' }
      }
    }
  }
});
// Transactions collection
db.createCollection('transactions', {
  validator: {
    $jsonSchema: {
      bsonType: 'object',
      required: ['_id', 'amount', 'type', 'category_id', 'user_id', 'family_id', 'date', 'created_at', 'updated_at'],
      properties: {
        _id: { bsonType: 'string' },
        amount: { bsonType: 'double', minimum: 0 },
        type: { bsonType: 'string', enum: ['income', 'expense'] },
        description: { bsonType: 'string' },
        category_id: { bsonType: 'string' },
        user_id: { bsonType: 'string' },
        family_id: { bsonType: 'string' },
        date: { bsonType: 'date' },
        tags: { bsonType: 'array', items: { bsonType: 'string' } },
        created_at: { bsonType: 'date' },
        updated_at: { bsonType: 'date' }
      }
    }
  }
});
// Budgets collection
db.createCollection('budgets', {
  validator: {
    $jsonSchema: {
      bsonType: 'object',
      required: ['_id', 'name', 'amount', 'spent', 'period', 'family_id', 'start_date', 'end_date', 'is_active', 'created_at', 'updated_at'],
      properties: {
        _id: { bsonType: 'string' },
        name: { bsonType: 'string' },
        amount: { bsonType: 'double', minimum: 0 },
        spent: { bsonType: 'double', minimum: 0 },
        period: { bsonType: 'string', enum: ['weekly', 'monthly', 'yearly'] },
        category_id: { bsonType: 'string' },
        family_id: { bsonType: 'string' },
        start_date: { bsonType: 'date' },
        end_date: { bsonType: 'date' },
        is_active: { bsonType: 'bool' },
        created_at: { bsonType: 'date' },
        updated_at: { bsonType: 'date' }
      }
    }
  }
});
// Reports collection
db.createCollection('reports', {
  validator: {
    $jsonSchema: {
      bsonType: 'object',
      required: ['_id', 'name', 'type', 'family_id', 'created_at', 'updated_at'],
      properties: {
        _id: { bsonType: 'string' },
        name: { bsonType: 'string' },
        type: { bsonType: 'string' },
        family_id: { bsonType: 'string' },
        parameters: { bsonType: 'object' },
        created_at: { bsonType: 'date' },
        updated_at: { bsonType: 'date' }
      }
    }
  }
});
print('📊 Creating indexes for optimal performance...');
// Users indexes
db.users.createIndex({ email: 1 }, { unique: true });
db.users.createIndex({ family_id: 1 });
db.users.createIndex({ family_id: 1, role: 1 });
// Families indexes
db.families.createIndex({ name: 1 });
// Categories indexes
db.categories.createIndex({ family_id: 1 });
db.categories.createIndex({ family_id: 1, type: 1 });
db.categories.createIndex({ family_id: 1, is_active: 1 });
db.categories.createIndex({ parent_id: 1 });
// Transactions indexes
db.transactions.createIndex({ family_id: 1 });
db.transactions.createIndex({ family_id: 1, date: -1 });
db.transactions.createIndex({ family_id: 1, type: 1 });
db.transactions.createIndex({ family_id: 1, category_id: 1 });
db.transactions.createIndex({ family_id: 1, user_id: 1 });
db.transactions.createIndex({ date: -1 });
db.transactions.createIndex({ tags: 1 });
// Budgets indexes
db.budgets.createIndex({ family_id: 1 });
db.budgets.createIndex({ family_id: 1, is_active: 1 });
db.budgets.createIndex({ family_id: 1, period: 1 });
db.budgets.createIndex({ category_id: 1 });
db.budgets.createIndex({ start_date: 1, end_date: 1 });
// Reports indexes
db.reports.createIndex({ family_id: 1 });
db.reports.createIndex({ family_id: 1, type: 1 });
print('🏷️  Creating default categories...');
// Default expense categories
const defaultExpenseCategories = [
  { name: 'Продукты', color: '#4CAF50', icon: '🛒' },
  { name: 'Транспорт', color: '#2196F3', icon: '🚗' },
  { name: 'Здоровье', color: '#F44336', icon: '💊' },
  { name: 'Развлечения', color: '#9C27B0', icon: '🎬' },
  { name: 'Одежда', color: '#FF9800', icon: '👕' },
  { name: 'Коммунальные услуги', color: '#607D8B', icon: '💡' },
  { name: 'Образование', color: '#3F51B5', icon: '📚' },
  { name: 'Ресторан', color: '#E91E63', icon: '🍽️' },
  { name: 'Хобби', color: '#00BCD4', icon: '🎨' },
  { name: 'Подарки', color: '#8BC34A', icon: '🎁' },
  { name: 'Спорт', color: '#FF5722', icon: '⚽' },
  { name: 'Путешествия', color: '#795548', icon: '✈️' },
  { name: 'Домашние животные', color: '#FFC107', icon: '🐕' },
  { name: 'Красота', color: '#E1BEE7', icon: '💄' },
  { name: 'Другое', color: '#9E9E9E', icon: '📦' }
];
// Default income categories
const defaultIncomeCategories = [
  { name: 'Зарплата', color: '#4CAF50', icon: '💼' },
  { name: 'Фриланс', color: '#2196F3', icon: '💻' },
  { name: 'Бизнес', color: '#FF9800', icon: '🏢' },
  { name: 'Инвестиции', color: '#9C27B0', icon: '📈' },
  { name: 'Подарки', color: '#E91E63', icon: '🎁' },
  { name: 'Продажи', color: '#00BCD4', icon: '💰' },
  { name: 'Другое', color: '#9E9E9E', icon: '📦' }
];
// Generate demo UUIDs and insert default categories
function generateDemoUUID() {
  const timestamp = Date.now().toString(16);
  const random = Math.random().toString(16).substr(2, 8);
  return timestamp + '-' + random + '-demo-uuid';
}
const demoFamilyId = 'demo-family-' + generateDemoUUID();
const currentDate = new Date();
defaultExpenseCategories.forEach(category => {
  db.categories.insertOne({
    _id: generateDemoUUID(),
    name: category.name,
    type: 'expense',
    color: category.color,
    icon: category.icon,
    family_id: demoFamilyId,
    is_active: true,
    created_at: currentDate,
    updated_at: currentDate
  });
});
defaultIncomeCategories.forEach(category => {
  db.categories.insertOne({
    _id: generateDemoUUID(),
    name: category.name,
    type: 'income',
    color: category.color,
    icon: category.icon,
    family_id: demoFamilyId,
    is_active: true,
    created_at: currentDate,
    updated_at: currentDate
  });
});
// Create compound indexes for complex queries
db.transactions.createIndex({ 
  family_id: 1, 
  date: -1, 
  type: 1 
}, { name: 'family_date_type_idx' });
db.transactions.createIndex({ 
  family_id: 1, 
  category_id: 1, 
  date: -1 
}, { name: 'family_category_date_idx' });
// Create sessions collection with TTL index
db.createCollection('sessions');
db.sessions.createIndex({ 
  expires_at: 1 
}, { 
  expireAfterSeconds: 0,
  name: 'session_ttl_idx'
});
// Create text indexes for search
db.transactions.createIndex({ 
  description: 'text',
  tags: 'text'
}, { 
  name: 'transaction_search_idx',
  default_language: 'russian'
});
print('✅ MongoDB initialization completed successfully!');
print('📋 Summary: Database ready with 6 collections, 25+ indexes, and default categories');
