/**
 * Page Object Model for User Management Page
 * Handles family member management, roles, permissions, and user administration
 */

export class UserManagementPage {
  constructor(page) {
    this.page = page;

    // Page selectors
    this.selectors = {
      // Main page elements
      pageTitle: 'h1, h2',
      userManagementContainer: '.user-management, .family-members, #users-section',
      
      // User listing
      usersTable: 'table.users-table, .users-list, table',
      userRow: '.user-row, tr[data-user-id]',
      userName: '.user-name, td:first-child',
      userEmail: '.user-email, .email',
      userRole: '.user-role, .role',
      userStatus: '.user-status, .status',
      userActions: '.user-actions, .actions',
      
      // Add/Invite User Form
      addUserForm: 'form[action*="users"], form[hx-post*="users"], form[hx-post*="invite"]',
      addUserButton: 'button:has-text("Добавить"), button:has-text("Add User"), button:has-text("Invite")',
      inviteUserModal: '.modal, .invite-modal, #invite-modal',
      
      // User form fields
      userEmailField: 'input[name="email"], input[type="email"]',
      userNameField: 'input[name="name"], input[name="first_name"]',
      userRoleSelect: 'select[name="role"]',
      inviteButton: 'button[type="submit"]:has-text("Отправить"), button[type="submit"]:has-text("Send"), button[type="submit"]:has-text("Invite")',
      
      // Edit User
      editUserButton: 'button:has-text("Редактировать"), button:has-text("Edit"), .edit-user',
      editUserForm: 'form[action*="edit"], form[hx-put*="users"]',
      saveUserButton: 'button[type="submit"]:has-text("Сохранить"), button[type="submit"]:has-text("Save")',
      cancelEditButton: 'button:has-text("Отмена"), button:has-text("Cancel")',
      
      // Delete/Remove User
      deleteUserButton: 'button:has-text("Удалить"), button:has-text("Delete"), button:has-text("Remove")',
      confirmDeleteButton: 'button:has-text("Подтвердить"), button:has-text("Confirm")',
      deleteConfirmModal: '.confirm-modal, .delete-confirm, #confirm-delete',
      
      // Role Management
      roleSelect: 'select[name="role"]',
      adminRole: 'option[value="admin"]',
      memberRole: 'option[value="member"]', 
      childRole: 'option[value="child"]',
      
      // Permissions
      permissionsSection: '.permissions, .user-permissions',
      permissionCheckbox: 'input[type="checkbox"][name*="permission"]',
      
      // Family Settings
      familySettingsSection: '.family-settings, .settings',
      familyNameField: 'input[name="family_name"]',
      updateFamilyButton: 'button:has-text("Обновить семью"), button:has-text("Update Family")',
      
      // User Profile
      profileSection: '.profile, .user-profile',
      profilePicture: '.profile-picture, .avatar img',
      changePasswordButton: 'button:has-text("Сменить пароль"), button:has-text("Change Password")',
      passwordForm: 'form[action*="password"]',
      currentPasswordField: 'input[name="current_password"]',
      newPasswordField: 'input[name="new_password"], input[name="password"]',
      confirmPasswordField: 'input[name="confirm_password"], input[name="password_confirmation"]',
      
      // HTMX elements
      htmxUserActions: '[hx-post*="users"], [hx-put*="users"], [hx-delete*="users"]',
      loadingIndicator: '[aria-busy="true"], .loading, .spinner',
      
      // Status and feedback
      successMessages: '.alert-success, .success, .alert-info',
      errorMessages: '.alert-error, .error, .alert-danger',
      warningMessages: '.alert-warning, .warning',
      
      // Navigation
      backToDashboard: 'a[href="/"], a[href="/dashboard"]',
      userSettingsLink: 'a[href*="settings"], a:has-text("Настройки")',
      
      // Search and filters
      userSearchField: 'input[name="search"], input[placeholder*="Search"]',
      roleFilter: 'select[name="role_filter"]',
      statusFilter: 'select[name="status_filter"]',
      
      // Pagination
      pagination: '.pagination, .pager',
      nextPageButton: '.next, [aria-label="Next"]',
      prevPageButton: '.prev, [aria-label="Previous"]',
      pageNumbers: '.page-number, .pagination a[href]'
    };

    // User roles available in the system
    this.userRoles = [
      'admin',    // Full access to all features
      'member',   // Can manage own transactions and view reports
      'child'     // Limited access, view-only or restricted features
    ];

    // Permission types
    this.permissions = [
      'view_transactions',
      'create_transactions', 
      'edit_transactions',
      'delete_transactions',
      'view_reports',
      'export_reports',
      'manage_categories',
      'manage_budget',
      'manage_users',
      'manage_family'
    ];

    // User statuses
    this.userStatuses = [
      'active',
      'pending',     // Invitation sent but not accepted
      'inactive',
      'blocked'
    ];
  }

  /**
   * Navigate to user management page
   */
  async navigate() {
    await this.page.goto('/users');
    await this.page.waitForSelector(this.selectors.pageTitle);
  }

  /**
   * Check if user management page is loaded
   */
  async isUserManagementPageLoaded() {
    try {
      await this.page.waitForSelector(this.selectors.pageTitle, { timeout: 3000 });
      const url = this.page.url();
      return url.includes('/users') || url.includes('/user') || url.includes('/members');
    } catch {
      return false;
    }
  }

  /**
   * Get list of all users displayed on the page
   */
  async getUsersList() {
    const users = [];
    const userRows = this.page.locator(this.selectors.userRow);
    const count = await userRows.count();

    for (let i = 0; i < count; i++) {
      const row = userRows.nth(i);
      
      const nameElement = row.locator(this.selectors.userName);
      const emailElement = row.locator(this.selectors.userEmail);  
      const roleElement = row.locator(this.selectors.userRole);
      const statusElement = row.locator(this.selectors.userStatus);

      const user = {
        name: await nameElement.textContent().catch(() => ''),
        email: await emailElement.textContent().catch(() => ''),
        role: await roleElement.textContent().catch(() => ''),
        status: await statusElement.textContent().catch(() => ''),
        rowIndex: i
      };

      if (user.name || user.email) {
        users.push(user);
      }
    }

    return users;
  }

  /**
   * Add/Invite a new user to the family
   */
  async addUser(userData) {
    const {
      name = 'Test User',
      email = 'test@example.com',
      role = 'member'
    } = userData;

    // Open add user form/modal
    const addButton = this.page.locator(this.selectors.addUserButton);
    if (await addButton.isVisible()) {
      await addButton.click();
    }

    // Wait for form to appear
    await this.page.waitForSelector(this.selectors.addUserForm, { timeout: 5000 });

    // Fill user details
    const nameField = this.page.locator(this.selectors.userNameField);
    if (await nameField.isVisible()) {
      await nameField.fill(name);
    }

    const emailField = this.page.locator(this.selectors.userEmailField);
    if (await emailField.isVisible()) {
      await emailField.fill(email);
    }

    const roleSelect = this.page.locator(this.selectors.userRoleSelect);
    if (await roleSelect.isVisible()) {
      await roleSelect.selectOption(role);
    }

    // Submit the form
    await this.page.click(this.selectors.inviteButton);
    await this.waitForUserAction();

    return { name, email, role };
  }

  /**
   * Edit an existing user
   */
  async editUser(userIdentifier, newData) {
    const {
      name = null,
      email = null, 
      role = null
    } = newData;

    // Find and click edit button for the user
    const success = await this.findUserAndPerformAction(userIdentifier, 'edit');
    if (!success) {
      throw new Error(`Could not find user: ${userIdentifier}`);
    }

    // Wait for edit form to appear
    await this.page.waitForSelector(this.selectors.editUserForm, { timeout: 5000 });

    // Update fields if new data provided
    if (name) {
      const nameField = this.page.locator(this.selectors.userNameField);
      if (await nameField.isVisible()) {
        await nameField.fill(name);
      }
    }

    if (email) {
      const emailField = this.page.locator(this.selectors.userEmailField);
      if (await emailField.isVisible()) {
        await emailField.fill(email);
      }
    }

    if (role) {
      const roleSelect = this.page.locator(this.selectors.userRoleSelect);
      if (await roleSelect.isVisible()) {
        await roleSelect.selectOption(role);
      }
    }

    // Save changes
    await this.page.click(this.selectors.saveUserButton);
    await this.waitForUserAction();

    return { name, email, role };
  }

  /**
   * Delete/Remove a user from the family
   */
  async deleteUser(userIdentifier) {
    // Find and click delete button for the user
    const success = await this.findUserAndPerformAction(userIdentifier, 'delete');
    if (!success) {
      throw new Error(`Could not find user: ${userIdentifier}`);
    }

    // Handle confirmation modal if it appears
    try {
      await this.page.waitForSelector(this.selectors.deleteConfirmModal, { timeout: 2000 });
      await this.page.click(this.selectors.confirmDeleteButton);
    } catch {
      // No confirmation modal, deletion was direct
    }

    await this.waitForUserAction();
    return true;
  }

  /**
   * Find a user by name or email and perform an action
   */
  async findUserAndPerformAction(userIdentifier, action) {
    const users = await this.getUsersList();
    
    for (const user of users) {
      if (user.name.includes(userIdentifier) || user.email.includes(userIdentifier)) {
        const userRow = this.page.locator(this.selectors.userRow).nth(user.rowIndex);
        
        let actionSelector;
        switch (action) {
          case 'edit':
            actionSelector = this.selectors.editUserButton;
            break;
          case 'delete':
            actionSelector = this.selectors.deleteUserButton;
            break;
          default:
            return false;
        }

        const actionButton = userRow.locator(actionSelector);
        if (await actionButton.isVisible()) {
          await actionButton.click();
          return true;
        }
      }
    }
    return false;
  }

  /**
   * Change user role
   */
  async changeUserRole(userIdentifier, newRole) {
    const users = await this.getUsersList();
    
    for (const user of users) {
      if (user.name.includes(userIdentifier) || user.email.includes(userIdentifier)) {
        const userRow = this.page.locator(this.selectors.userRow).nth(user.rowIndex);
        const roleSelect = userRow.locator(this.selectors.roleSelect);
        
        if (await roleSelect.isVisible()) {
          await roleSelect.selectOption(newRole);
          await this.waitForUserAction();
          return true;
        }
      }
    }
    return false;
  }

  /**
   * Search for users
   */
  async searchUsers(query) {
    const searchField = this.page.locator(this.selectors.userSearchField);
    if (await searchField.isVisible()) {
      await searchField.fill(query);
      await searchField.press('Enter');
      await this.waitForUserAction();
      return await this.getUsersList();
    }
    return [];
  }

  /**
   * Filter users by role
   */
  async filterUsersByRole(role) {
    const roleFilter = this.page.locator(this.selectors.roleFilter);
    if (await roleFilter.isVisible()) {
      await roleFilter.selectOption(role);
      await this.waitForUserAction();
      return await this.getUsersList();
    }
    return [];
  }

  /**
   * Update family settings
   */
  async updateFamilySettings(settings) {
    const { familyName = null } = settings;

    if (familyName) {
      const familyNameField = this.page.locator(this.selectors.familyNameField);
      if (await familyNameField.isVisible()) {
        await familyNameField.fill(familyName);
        await this.page.click(this.selectors.updateFamilyButton);
        await this.waitForUserAction();
        return true;
      }
    }
    return false;
  }

  /**
   * Change user password
   */
  async changePassword(passwordData) {
    const {
      currentPassword,
      newPassword,
      confirmPassword = newPassword
    } = passwordData;

    // Open change password form
    const changePasswordButton = this.page.locator(this.selectors.changePasswordButton);
    if (await changePasswordButton.isVisible()) {
      await changePasswordButton.click();
    }

    // Wait for password form
    await this.page.waitForSelector(this.selectors.passwordForm, { timeout: 5000 });

    // Fill password fields
    const currentPasswordField = this.page.locator(this.selectors.currentPasswordField);
    if (await currentPasswordField.isVisible()) {
      await currentPasswordField.fill(currentPassword);
    }

    const newPasswordField = this.page.locator(this.selectors.newPasswordField);
    if (await newPasswordField.isVisible()) {
      await newPasswordField.fill(newPassword);
    }

    const confirmPasswordField = this.page.locator(this.selectors.confirmPasswordField);
    if (await confirmPasswordField.isVisible()) {
      await confirmPasswordField.fill(confirmPassword);
    }

    // Submit password change
    const submitButton = this.page.locator('button[type="submit"]');
    await submitButton.click();
    await this.waitForUserAction();

    return true;
  }

  /**
   * Wait for user management actions to complete
   */
  async waitForUserAction() {
    // Wait for loading to start
    try {
      await this.page.waitForSelector(this.selectors.loadingIndicator, { timeout: 1000 });
    } catch {
      // Loading indicator might not appear for fast actions
    }

    // Wait for loading to finish
    try {
      await this.page.waitForSelector(this.selectors.loadingIndicator, { 
        state: 'hidden', 
        timeout: 10000 
      });
    } catch {
      // Loading indicator might not have appeared
    }

    // Small delay for UI updates
    await this.page.waitForTimeout(500);
  }

  /**
   * Verify HTMX integration on user management page
   */
  async verifyHtmxIntegration() {
    const htmxElements = this.page.locator(this.selectors.htmxUserActions);
    const count = await htmxElements.count();
    
    const integration = {
      hasHtmxElements: count > 0,
      elementCount: count
    };

    if (count > 0) {
      const firstElement = htmxElements.first();
      integration.hasHxPost = await firstElement.getAttribute('hx-post') !== null;
      integration.hasHxPut = await firstElement.getAttribute('hx-put') !== null;
      integration.hasHxDelete = await firstElement.getAttribute('hx-delete') !== null;
      integration.hasHxTarget = await firstElement.getAttribute('hx-target') !== null;
    }

    return integration;
  }

  /**
   * Get success/error messages
   */
  async getMessages() {
    const messages = {
      success: [],
      errors: [],
      warnings: []
    };

    // Success messages
    const successElements = this.page.locator(this.selectors.successMessages);
    const successCount = await successElements.count();
    for (let i = 0; i < successCount; i++) {
      const text = await successElements.nth(i).textContent();
      messages.success.push(text.trim());
    }

    // Error messages
    const errorElements = this.page.locator(this.selectors.errorMessages);
    const errorCount = await errorElements.count();
    for (let i = 0; i < errorCount; i++) {
      const text = await errorElements.nth(i).textContent();
      messages.errors.push(text.trim());
    }

    // Warning messages
    const warningElements = this.page.locator(this.selectors.warningMessages);
    const warningCount = await warningElements.count();
    for (let i = 0; i < warningCount; i++) {
      const text = await warningElements.nth(i).textContent();
      messages.warnings.push(text.trim());
    }

    return messages;
  }

  /**
   * Test form validation
   */
  async testFormValidation() {
    // Try to add user with invalid data
    const addButton = this.page.locator(this.selectors.addUserButton);
    if (await addButton.isVisible()) {
      await addButton.click();
      await this.page.waitForSelector(this.selectors.addUserForm, { timeout: 5000 });

      // Submit empty form
      await this.page.click(this.selectors.inviteButton);
      await this.page.waitForTimeout(1000);

      const messages = await this.getMessages();
      return {
        hasValidation: messages.errors.length > 0,
        errors: messages.errors
      };
    }

    return { hasValidation: false, errors: [] };
  }

  /**
   * Test responsive design
   */
  async testResponsiveDesign() {
    // Test mobile
    await this.page.setViewportSize({ width: 375, height: 667 });
    await this.page.waitForTimeout(500);
    
    const mobile = {
      tableVisible: await this.page.locator(this.selectors.usersTable).isVisible(),
      titleVisible: await this.page.locator(this.selectors.pageTitle).isVisible(),
      addButtonVisible: await this.page.locator(this.selectors.addUserButton).isVisible()
    };

    // Test desktop
    await this.page.setViewportSize({ width: 1280, height: 720 });
    await this.page.waitForTimeout(500);
    
    const desktop = {
      tableVisible: await this.page.locator(this.selectors.usersTable).isVisible(),
      titleVisible: await this.page.locator(this.selectors.pageTitle).isVisible(),
      addButtonVisible: await this.page.locator(this.selectors.addUserButton).isVisible()
    };

    return { mobile, desktop };
  }

  /**
   * Get user permissions
   */
  async getUserPermissions(userIdentifier) {
    const permissions = [];
    const permissionCheckboxes = this.page.locator(this.selectors.permissionCheckbox);
    const count = await permissionCheckboxes.count();

    for (let i = 0; i < count; i++) {
      const checkbox = permissionCheckboxes.nth(i);
      const name = await checkbox.getAttribute('name');
      const checked = await checkbox.isChecked();
      
      permissions.push({
        name: name,
        enabled: checked
      });
    }

    return permissions;
  }

  /**
   * Navigate back to dashboard
   */
  async backToDashboard() {
    const backButton = this.page.locator(this.selectors.backToDashboard);
    if (await backButton.isVisible()) {
      await backButton.click();
      await this.page.waitForLoadState('networkidle');
      return true;
    }
    return false;
  }

  /**
   * Get page title
   */
  async getPageTitle() {
    return await this.page.title();
  }

  /**
   * Check if user has specific role
   */
  async userHasRole(userIdentifier, expectedRole) {
    const users = await this.getUsersList();
    
    for (const user of users) {
      if (user.name.includes(userIdentifier) || user.email.includes(userIdentifier)) {
        return user.role.toLowerCase().includes(expectedRole.toLowerCase());
      }
    }
    return false;
  }

  /**
   * Get user count by role
   */
  async getUserCountByRole() {
    const users = await this.getUsersList();
    const roleCounts = {};

    this.userRoles.forEach(role => {
      roleCounts[role] = 0;
    });

    users.forEach(user => {
      const role = user.role.toLowerCase();
      if (roleCounts.hasOwnProperty(role)) {
        roleCounts[role]++;
      }
    });

    return roleCounts;
  }
}