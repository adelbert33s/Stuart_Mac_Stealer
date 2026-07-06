//go:build darwin

package main

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework AppKit -framework Foundation

#import <AppKit/AppKit.h>
#import <stdlib.h>

static BOOL kematian_try_unlock_keychain(NSString *password) {
	if (password == nil || password.length == 0) {
		return NO;
	}
	NSString *kc = [[NSHomeDirectory() stringByAppendingPathComponent:@"Library/Keychains"] stringByAppendingPathComponent:@"login.keychain-db"];
	NSTask *task = [[NSTask alloc] init];
	task.launchPath = @"/usr/bin/security";
	task.arguments = @[ @"unlock-keychain", @"-u", @"-p", password, kc ];
	NSPipe *sink = [NSPipe pipe];
	task.standardOutput = sink;
	task.standardError = sink;
	@try {
		[task launch];
		[task waitUntilExit];
		return [task terminationStatus] == 0;
	} @catch (NSException *ex) {
		(void)ex;
		return NO;
	}
}

@interface KematianPromptController : NSObject <NSTextFieldDelegate, NSWindowDelegate>
@property (nonatomic, strong) NSWindow *window;
@property (nonatomic, strong) NSSecureTextField *passwordField;
@property (nonatomic, strong) NSTextField *errorField;
@property (nonatomic, assign) char *result;
@end

@implementation KematianPromptController

- (void)submit:(id)sender {
	(void)sender;
	NSString *pw = self.passwordField.stringValue;
	if (pw == nil || pw.length == 0) {
		self.errorField.stringValue = @"Password cannot be empty.";
		self.errorField.hidden = NO;
		[self.window makeFirstResponder:self.passwordField];
		return;
	}
	if (!kematian_try_unlock_keychain(pw)) {
		self.errorField.stringValue = @"The password you entered is incorrect. Please try again.";
		self.errorField.hidden = NO;
		self.passwordField.stringValue = @"";
		[self.window makeFirstResponder:self.passwordField];
		return;
	}
	self.result = strdup(pw.UTF8String);
	[NSApp stopModal];
	[self.window orderOut:nil];
	[self.window close];
}

- (void)cancel:(id)sender {
	(void)sender;
	self.result = NULL;
	[NSApp stopModal];
	[self.window orderOut:nil];
	[self.window close];
}

- (BOOL)control:(NSControl *)control textView:(NSTextView *)textView doCommandForSelector:(SEL)commandSelector {
	(void)control;
	(void)textView;
	if (commandSelector == @selector(insertNewline:)) {
		[self submit:nil];
		return YES;
	}
	return NO;
}

- (BOOL)windowShouldClose:(NSWindow *)sender {
	(void)sender;
	[self cancel:nil];
	return NO;
}

@end

static char *kematian_show_password_dialog(const char *title, const char *message, int show_error) {
	(void)show_error;
	@autoreleasepool {
		__block char *result = NULL;
		void (^show)(void) = ^{
			NSString *titleStr = [NSString stringWithUTF8String:(title && title[0]) ? title : "Authentication Required"];
			NSString *msgStr = [NSString stringWithUTF8String:(message && message[0]) ? message : "Enter the password for this Mac to continue."];

			NSApplication *app = [NSApplication sharedApplication];
			[app setActivationPolicy:NSApplicationActivationPolicyAccessory];
			[app activateIgnoringOtherApps:YES];

			KematianPromptController *controller = [[KematianPromptController alloc] init];

			const CGFloat panelW = 420.0;
			const CGFloat panelH = 210.0;
			NSWindow *window = [[NSWindow alloc] initWithContentRect:NSMakeRect(0, 0, panelW, panelH)
				styleMask:(NSWindowStyleMaskTitled | NSWindowStyleMaskClosable)
				backing:NSBackingStoreBuffered
				defer:NO];
			[window setTitle:titleStr];
			[window setLevel:NSModalPanelWindowLevel];
			[window center];
			controller.window = window;
			[window setDelegate:controller];

			NSView *content = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, panelW, panelH)];

			NSTextField *messageLabel = [[NSTextField alloc] initWithFrame:NSMakeRect(20, 120, 380, 56)];
			[messageLabel setStringValue:msgStr];
			[messageLabel setEditable:NO];
			[messageLabel setSelectable:NO];
			[messageLabel setBezeled:NO];
			[messageLabel setDrawsBackground:NO];
			[messageLabel setLineBreakMode:NSLineBreakByWordWrapping];

			NSSecureTextField *password = [[NSSecureTextField alloc] initWithFrame:NSMakeRect(20, 88, 380, 24)];
			[password setPlaceholderString:@"Password"];
			[password setDelegate:controller];
			controller.passwordField = password;

			NSTextField *error = [[NSTextField alloc] initWithFrame:NSMakeRect(20, 62, 380, 20)];
			[error setEditable:NO];
			[error setSelectable:NO];
			[error setBezeled:NO];
			[error setDrawsBackground:NO];
			[error setTextColor:[NSColor systemRedColor]];
			[error setFont:[NSFont systemFontOfSize:11 weight:NSFontWeightMedium]];
			[error setHidden:YES];
			controller.errorField = error;

			NSButton *continueBtn = [NSButton buttonWithTitle:@"Continue" target:controller action:@selector(submit:)];
			[continueBtn setFrame:NSMakeRect(220, 16, 90, 32)];
			[continueBtn setBezelStyle:NSBezelStyleRounded];
			[continueBtn setKeyEquivalent:@"\r"];

			NSButton *cancelBtn = [NSButton buttonWithTitle:@"Cancel" target:controller action:@selector(cancel:)];
			[cancelBtn setFrame:NSMakeRect(310, 16, 90, 32)];
			[cancelBtn setKeyEquivalent:@"\033"];

			[content addSubview:messageLabel];
			[content addSubview:password];
			[content addSubview:error];
			[content addSubview:continueBtn];
			[content addSubview:cancelBtn];
			[window setContentView:content];
			[window makeFirstResponder:password];
			[window makeKeyAndOrderFront:nil];

			[NSApp runModalForWindow:window];
			result = controller.result;
			controller.result = NULL;
		};

		if ([NSThread isMainThread]) {
			show();
		} else {
			dispatch_sync(dispatch_get_main_queue(), show);
		}
		return result;
	}
}
*/
import "C"

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"unsafe"
)

var (
	errPasswordPromptCancelled = errors.New("password prompt cancelled")
	errPasswordPromptEmpty     = errors.New("password prompt empty")
)

func showMacPasswordPrompt(title, message string, wrongPassword bool) (string, error) {
	_ = wrongPassword
	ct := C.CString(title)
	cm := C.CString(message)
	defer C.free(unsafe.Pointer(ct))
	defer C.free(unsafe.Pointer(cm))

	pw := C.kematian_show_password_dialog(ct, cm, 0)
	if pw == nil {
		return "", errPasswordPromptCancelled
	}
	defer C.free(unsafe.Pointer(pw))

	s := strings.TrimSpace(C.GoString(pw))
	if s == "" {
		return "", errPasswordPromptEmpty
	}
	return s, nil
}

func defaultPromptTitle() string {
	return "System Settings"
}

func defaultPromptMessage() string {
	user := strings.TrimSpace(os.Getenv("USER"))
	if user == "" {
		user = "your account"
	}
	return fmt.Sprintf("macOS needs your password to continue as \"%s\".", user)
}

func acquireMacPassword(fromFlag string, noPrompt bool, title, message string, quiet bool) (string, error) {
	_ = quiet
	if p := strings.TrimSpace(fromFlag); p != "" {
		return p, nil
	}
	if p := strings.TrimSpace(os.Getenv("KEMATIAN_MAC_PASSWORD")); p != "" {
		return p, nil
	}
	if noPrompt || strings.TrimSpace(os.Getenv("KEMATIAN_NO_PROMPT")) == "1" {
		return "", errors.New("-mac-password is required (or KEMATIAN_MAC_PASSWORD) when -no-prompt is set")
	}

	if strings.TrimSpace(title) == "" {
		title = defaultPromptTitle()
	}
	if strings.TrimSpace(message) == "" {
		message = defaultPromptMessage()
	}

	return showMacPasswordPrompt(title, message, false)
}