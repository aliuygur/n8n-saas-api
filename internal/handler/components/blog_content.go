package components

// GetBlogContent returns the full HTML content for a blog post by slug
func GetBlogContent(slug string) string {
	content := map[string]string{
		"what-is-n8n-workflow-automation": `
<article class="prose prose-invert prose-lg max-w-none">
	<p class="lead text-xl text-gray-300 mb-8">n8n is a powerful, open-source workflow automation tool that helps you connect different applications and services to automate repetitive tasks. Whether you're a developer, marketer, or business owner, n8n enables you to build complex workflows without extensive coding knowledge.</p>

	<h2>What Makes n8n Special?</h2>
	<p>Unlike many automation tools, n8n is <strong>fair-code licensed</strong>, meaning the source code is available for everyone to view and self-host. This gives you complete control over your data and automation workflows.</p>

	<h3>Key Features of n8n</h3>
	<ul>
		<li><strong>400+ Integrations</strong> - Connect with popular apps like Gmail, Slack, Google Sheets, Airtable, and more</li>
		<li><strong>Visual Workflow Editor</strong> - Build automations with an intuitive drag-and-drop interface</li>
		<li><strong>Self-Hostable</strong> - Deploy on your own infrastructure for complete data control</li>
		<li><strong>Custom Code Support</strong> - Write JavaScript when you need advanced logic</li>
		<li><strong>Active Community</strong> - Thousands of developers contributing and sharing workflows</li>
	</ul>

	<h2>How Does n8n Work?</h2>
	<p>n8n works by creating "workflows" - a series of connected "nodes" that perform specific actions. Here's a simple example:</p>
	<ol>
		<li><strong>Trigger</strong> - Something happens (e.g., new email arrives)</li>
		<li><strong>Process</strong> - Extract information from the email</li>
		<li><strong>Action</strong> - Save data to Google Sheets</li>
		<li><strong>Notify</strong> - Send a Slack message</li>
	</ol>

	<h2>Common Use Cases</h2>
	<p>Businesses use n8n to automate various tasks:</p>
	<ul>
		<li>Syncing data between CRM and marketing tools</li>
		<li>Automating customer support workflows</li>
		<li>Processing and organizing incoming data</li>
		<li>Social media management and posting</li>
		<li>Data backup and synchronization</li>
	</ul>

	<h2>Self-Hosted vs Cloud</h2>
	<p>n8n offers two deployment options:</p>
	<p><strong>Self-Hosted:</strong> You manage the infrastructure, giving you complete control and data privacy. Great for security-conscious organizations.</p>
	<p><strong>n8n Cloud:</strong> Fully managed by the n8n team, perfect for quick setup without infrastructure management.</p>
	<p>With <strong>ranx.cloud</strong>, you get the best of both worlds: the control of self-hosting with the ease of managed infrastructure.</p>

	<h2>Getting Started with n8n</h2>
	<p>Ready to try n8n? With ranx.cloud, you can deploy your own n8n instance in under 2 minutes:</p>
	<ol>
		<li>Sign up for a free 3-day trial</li>
		<li>Choose your subdomain</li>
		<li>Start building workflows immediately</li>
	</ol>

	<div class="bg-indigo-900/30 border border-indigo-500/30 rounded-lg p-6 my-8">
		<h3 class="text-white mt-0">Try n8n on ranx.cloud</h3>
		<p class="mb-4">Deploy your managed n8n instance in minutes. No DevOps experience required.</p>
		<a href="/pricing" class="inline-block bg-indigo-600 text-white px-6 py-3 rounded-lg hover:bg-indigo-500 transition-colors font-semibold">Start Free Trial</a>
	</div>
</article>
`,

		"n8n-vs-zapier-comparison": `
<article class="prose prose-invert prose-lg max-w-none">
	<p class="lead text-xl text-gray-300 mb-8">Choosing between n8n and Zapier? Both are excellent automation platforms, but they serve different needs. This comprehensive comparison will help you decide which tool is right for your workflow automation requirements.</p>

	<h2>Overview</h2>
	<p><strong>Zapier</strong> is the market leader in no-code automation, known for its ease of use and extensive app ecosystem. <strong>n8n</strong> is an open-source alternative that offers more flexibility and control, especially for technical users.</p>

	<h2>Feature Comparison</h2>

	<h3>Pricing</h3>
	<p><strong>Zapier:</strong> Starts at $29.99/month for 750 tasks. Premium plans can cost $103.50/month or more.</p>
	<p><strong>n8n Cloud:</strong> Starts at €20/month for 2,500 executions.</p>
	<p><strong>Self-Hosted n8n (via ranx.cloud):</strong> Just $9/month with unlimited executions.</p>

	<h3>Integrations</h3>
	<p><strong>Zapier:</strong> 6,000+ app integrations</p>
	<p><strong>n8n:</strong> 400+ integrations, plus the ability to create custom integrations</p>

	<h3>Customization</h3>
	<p><strong>Zapier:</strong> Limited customization through built-in formatters and filters</p>
	<p><strong>n8n:</strong> Full JavaScript support for custom logic, HTTP requests, and data manipulation</p>

	<h3>Data Privacy</h3>
	<p><strong>Zapier:</strong> Data flows through Zapier's servers</p>
	<p><strong>n8n (Self-Hosted):</strong> Complete control - your data never leaves your infrastructure</p>

	<h2>When to Choose Zapier</h2>
	<ul>
		<li>You need the simplest possible setup</li>
		<li>You require integrations with niche applications</li>
		<li>Your team is completely non-technical</li>
		<li>You prefer a fully managed cloud solution</li>
	</ul>

	<h2>When to Choose n8n</h2>
	<ul>
		<li>You want unlimited executions at a fixed cost</li>
		<li>Data privacy and control are critical</li>
		<li>You need custom integrations or complex logic</li>
		<li>You want to avoid vendor lock-in with open-source software</li>
		<li>Your workflows require advanced data transformation</li>
	</ul>

	<h2>Cost Analysis Example</h2>
	<p>Let's say you run 10,000 workflow executions per month:</p>
	<ul>
		<li><strong>Zapier:</strong> ~$103.50/month (Professional plan)</li>
		<li><strong>n8n Cloud:</strong> ~€50/month</li>
		<li><strong>ranx.cloud:</strong> $9/month (unlimited executions)</li>
	</ul>

	<h2>Migration from Zapier to n8n</h2>
	<p>Many users successfully migrate from Zapier to n8n. While there's no automatic migration tool, most workflows can be rebuilt in n8n within a few hours, often with improved performance and more features.</p>

	<div class="bg-green-900/30 border border-green-500/30 rounded-lg p-6 my-8">
		<h3 class="text-white mt-0">Save Money with n8n on ranx.cloud</h3>
		<p class="mb-4">Run unlimited workflows for just $9/month. Perfect for businesses looking to reduce automation costs.</p>
		<a href="/create-instance" class="inline-block bg-green-600 text-white px-6 py-3 rounded-lg hover:bg-green-500 transition-colors font-semibold">Deploy n8n Now</a>
	</div>
</article>
`,

		"getting-started-with-n8n": `
<article class="prose prose-invert prose-lg max-w-none">
	<p class="lead text-xl text-gray-300 mb-8">New to n8n? This beginner-friendly guide will walk you through creating your first workflow in just 10 minutes. No coding experience required!</p>

	<h2>Prerequisites</h2>
	<p>Before we begin, you'll need:</p>
	<ul>
		<li>An n8n instance (deploy one on ranx.cloud in 2 minutes)</li>
		<li>A Gmail account (for our example workflow)</li>
		<li>A Slack workspace (optional, but recommended)</li>
	</ul>

	<h2>Step 1: Access Your n8n Instance</h2>
	<p>Once you've deployed n8n on ranx.cloud, you'll receive your unique URL (e.g., yourname.ranx.cloud). Open it in your browser to access the n8n interface.</p>

	<h2>Step 2: Create Your First Workflow</h2>
	<p>Click the <strong>"Create Workflow"</strong> button. You'll see a blank canvas with a single "+" button. This is where the magic happens!</p>

	<h3>Understanding Nodes</h3>
	<p>In n8n, workflows are built using "nodes" - individual blocks that perform specific actions:</p>
	<ul>
		<li><strong>Trigger Nodes</strong> - Start the workflow (e.g., "when email arrives")</li>
		<li><strong>Action Nodes</strong> - Perform tasks (e.g., "send Slack message")</li>
		<li><strong>Logic Nodes</strong> - Add conditions and transformations</li>
	</ul>

	<h2>Step 3: Add a Trigger</h2>
	<p>Let's create a workflow that monitors Gmail for new emails:</p>
	<ol>
		<li>Click the "+" button</li>
		<li>Search for "Gmail Trigger"</li>
		<li>Select "Gmail Trigger" node</li>
		<li>Click "Connect my account" and authorize Gmail</li>
		<li>Set "Event" to "Message Received"</li>
	</ol>

	<h2>Step 4: Add an Action</h2>
	<p>Now, let's send a Slack notification when an email arrives:</p>
	<ol>
		<li>Click the "+" on the Gmail Trigger node</li>
		<li>Search for "Slack"</li>
		<li>Select "Slack" node</li>
		<li>Connect your Slack account</li>
		<li>Choose your channel</li>
		<li>Write your message (use expressions to include email data)</li>
	</ol>

	<h3>Using Expressions</h3>
	<p>n8n's expressions let you use data from previous nodes. For example:</p>
	<pre class="bg-gray-900 p-4 rounded">New email from: {{ $json.from }}</pre>

	<h2>Step 5: Test Your Workflow</h2>
	<p>Before activating, test your workflow:</p>
	<ol>
		<li>Click "Execute Workflow" in the top right</li>
		<li>Send a test email to your Gmail</li>
		<li>Check if the Slack message appears</li>
		<li>Review the data passed between nodes</li>
	</ol>

	<h2>Step 6: Activate Your Workflow</h2>
	<p>Once testing is successful, toggle the "Active" switch in the top right. Your workflow now runs automatically!</p>

	<h2>Next Steps</h2>
	<p>Now that you've created your first workflow, try these:</p>
	<ul>
		<li>Add a filter to only notify for important emails</li>
		<li>Save email data to Google Sheets</li>
		<li>Create a workflow that posts to multiple channels</li>
		<li>Explore the 400+ integrations available</li>
	</ul>

	<div class="bg-blue-900/30 border border-blue-500/30 rounded-lg p-6 my-8">
		<h3 class="text-white mt-0">Ready to Build More Workflows?</h3>
		<p class="mb-4">Get your own n8n instance on ranx.cloud with 3-day free trial. No credit card required.</p>
		<a href="/login" class="inline-block bg-blue-600 text-white px-6 py-3 rounded-lg hover:bg-blue-500 transition-colors font-semibold">Start Building</a>
	</div>
</article>
`,

		"n8n-use-cases-examples": `
<article class="prose prose-invert prose-lg max-w-none">
	<p class="lead text-xl text-gray-300 mb-8">Discover how businesses are using n8n to automate repetitive tasks and save hundreds of hours every month. Here are 10 real-world use cases you can implement today.</p>

	<h2>1. Lead Management Automation</h2>
	<p><strong>Workflow:</strong> New form submission → Validate data → Add to CRM → Send welcome email → Notify sales team</p>
	<p><strong>Time Saved:</strong> ~2 hours per day</p>
	<p>Automatically capture leads from your website, validate the information, add them to your CRM, send a personalized welcome email, and notify your sales team on Slack.</p>

	<h2>2. Social Media Cross-Posting</h2>
	<p><strong>Workflow:</strong> New blog post → Extract content → Post to Twitter → Post to LinkedIn → Post to Facebook</p>
	<p><strong>Time Saved:</strong> ~30 minutes per post</p>
	<p>Publish once, distribute everywhere. Automatically share your content across all social media platforms when you publish a new blog post.</p>

	<h2>3. Customer Support Ticket Routing</h2>
	<p><strong>Workflow:</strong> New support email → Analyze content → Assign to specialist → Create ticket → Update status board</p>
	<p><strong>Time Saved:</strong> ~1 hour per day</p>
	<p>Route support requests to the right team member based on keywords, urgency, or customer tier. Keep your project management tool updated automatically.</p>

	<h2>4. E-commerce Order Processing</h2>
	<p><strong>Workflow:</strong> New order → Send confirmation → Add to inventory system → Generate invoice → Schedule fulfillment</p>
	<p><strong>Time Saved:</strong> ~3 hours per day</p>
	<p>Streamline your order fulfillment process by connecting your e-commerce platform with inventory management, accounting, and shipping systems.</p>

	<h2>5. Content Backup & Archiving</h2>
	<p><strong>Workflow:</strong> Schedule trigger → Fetch new content → Upload to cloud storage → Update index → Send summary report</p>
	<p><strong>Time Saved:</strong> ~2 hours per week</p>
	<p>Automatically backup your blog posts, social media content, or important documents to multiple cloud storage providers.</p>

	<h2>6. Employee Onboarding</h2>
	<p><strong>Workflow:</strong> New hire form → Create accounts → Send welcome package → Schedule training → Add to calendar</p>
	<p><strong>Time Saved:</strong> ~4 hours per new hire</p>
	<p>Automate the entire onboarding process from account creation to training scheduling, ensuring no steps are missed.</p>

	<h2>7. Data Synchronization</h2>
	<p><strong>Workflow:</strong> Schedule trigger → Fetch from Database A → Transform data → Update Database B → Log changes</p>
	<p><strong>Time Saved:</strong> ~5 hours per week</p>
	<p>Keep your databases in sync across different platforms, ensuring data consistency without manual CSV exports and imports.</p>

	<h2>8. Marketing Campaign Tracking</h2>
	<p><strong>Workflow:</strong> New campaign launch → Track metrics → Calculate ROI → Update dashboard → Send weekly reports</p>
	<p><strong>Time Saved:</strong> ~3 hours per week</p>
	<p>Automatically collect campaign data from various sources, calculate key metrics, and generate reports for stakeholders.</p>

	<h2>9. Invoice Generation & Payment Reminders</h2>
	<p><strong>Workflow:</strong> Schedule trigger → Generate invoices → Send to clients → Track payments → Send reminders for overdue</p>
	<p><strong>Time Saved:</strong> ~4 hours per month</p>
	<p>Automate your entire invoicing process including generation, distribution, payment tracking, and follow-ups.</p>

	<h2>10. Website Monitoring & Alerts</h2>
	<p><strong>Workflow:</strong> Schedule check → Monitor website → Detect issues → Send alerts → Create incident ticket</p>
	<p><strong>Time Saved:</strong> Prevents costly downtime</p>
	<p>Monitor your website's uptime, performance, and SSL certificates. Get instant alerts when issues are detected.</p>

	<h2>How to Get Started</h2>
	<p>All these workflows can be built in n8n with minimal technical knowledge. Start with a simple automation and gradually build more complex workflows as you learn.</p>

	<div class="bg-purple-900/30 border border-purple-500/30 rounded-lg p-6 my-8">
		<h3 class="text-white mt-0">Start Automating Today</h3>
		<p class="mb-4">Deploy your own n8n instance on ranx.cloud and start building these automations in minutes.</p>
		<a href="/pricing" class="inline-block bg-purple-600 text-white px-6 py-3 rounded-lg hover:bg-purple-500 transition-colors font-semibold">View Pricing</a>
	</div>
</article>
`,

		"self-hosted-n8n-vs-cloud": `
<article class="prose prose-invert prose-lg max-w-none">
	<p class="lead text-xl text-gray-300 mb-8">Deciding between self-hosted n8n and n8n Cloud? This guide compares both options to help you make the best choice for your automation needs.</p>

	<h2>What's the Difference?</h2>
	<p><strong>n8n Cloud</strong> is a fully managed service run by the n8n team. You don't worry about servers, updates, or infrastructure.</p>
	<p><strong>Self-Hosted n8n</strong> means you deploy n8n on your own infrastructure (or use a managed service like ranx.cloud), giving you complete control.</p>

	<h2>Comparison Table</h2>
	<div class="overflow-x-auto">
		<table class="min-w-full">
			<thead>
				<tr>
					<th>Feature</th>
					<th>Self-Hosted</th>
					<th>n8n Cloud</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td>Setup Time</td>
					<td>2 minutes (ranx.cloud)</td>
					<td>Instant</td>
				</tr>
				<tr>
					<td>Monthly Cost</td>
					<td>$9 (ranx.cloud)</td>
					<td>€20+</td>
				</tr>
				<tr>
					<td>Execution Limits</td>
					<td>Unlimited</td>
					<td>Limited by plan</td>
				</tr>
				<tr>
					<td>Data Privacy</td>
					<td>Full control</td>
					<td>Managed by n8n</td>
				</tr>
				<tr>
					<td>Updates</td>
					<td>Automatic (ranx.cloud)</td>
					<td>Automatic</td>
				</tr>
				<tr>
					<td>Custom Nodes</td>
					<td>Yes</td>
					<td>Limited</td>
				</tr>
				<tr>
					<td>Infrastructure Control</td>
					<td>Full</td>
					<td>None</td>
				</tr>
			</tbody>
		</table>
	</div>

	<h2>Benefits of Self-Hosted n8n</h2>

	<h3>1. Cost Efficiency</h3>
	<p>With self-hosting via ranx.cloud, you pay a flat $9/month with unlimited executions. As your automation grows, you save significantly compared to execution-based pricing.</p>

	<h3>2. Data Privacy & Security</h3>
	<p>Your data stays in your infrastructure. Perfect for healthcare, finance, or any industry with strict compliance requirements.</p>

	<h3>3. Customization</h3>
	<p>Install custom nodes, modify the source code, or integrate with internal systems that aren't publicly accessible.</p>

	<h3>4. No Vendor Lock-In</h3>
	<p>You own your automation infrastructure. If you ever want to move, you can export everything and redeploy anywhere.</p>

	<h2>Benefits of n8n Cloud</h2>

	<h3>1. Zero Maintenance</h3>
	<p>The n8n team handles all infrastructure, updates, and scaling. You just build workflows.</p>

	<h3>2. Enterprise Support</h3>
	<p>Direct support from the n8n team with guaranteed response times.</p>

	<h3>3. Instant Scaling</h3>
	<p>Automatically scales to handle traffic spikes without any configuration.</p>

	<h2>The Middle Ground: Managed Self-Hosting</h2>
	<p>Services like <strong>ranx.cloud</strong> offer the best of both worlds:</p>
	<ul>
		<li>✅ Easy deployment (2 minutes)</li>
		<li>✅ Automatic updates</li>
		<li>✅ SSL certificates included</li>
		<li>✅ Your own cloud infrastructure</li>
		<li>✅ Full data control</li>
		<li>✅ Fixed cost ($9/month)</li>
		<li>✅ Unlimited executions</li>
	</ul>

	<h2>Which Should You Choose?</h2>

	<h3>Choose Self-Hosted (via ranx.cloud) if:</h3>
	<ul>
		<li>You run high-volume workflows (>10,000/month)</li>
		<li>Data privacy is critical</li>
		<li>You want predictable costs</li>
		<li>You need custom integrations</li>
		<li>You want to avoid execution limits</li>
	</ul>

	<h3>Choose n8n Cloud if:</h3>
	<ul>
		<li>You want absolute zero maintenance</li>
		<li>You need enterprise support guarantees</li>
		<li>Your workflows are low-volume</li>
		<li>You prefer official n8n infrastructure</li>
	</ul>

	<div class="bg-indigo-900/30 border border-indigo-500/30 rounded-lg p-6 my-8">
		<h3 class="text-white mt-0">Get Started with Self-Hosted n8n</h3>
		<p class="mb-4">Deploy your managed n8n instance in under 2 minutes. 3-day free trial, no credit card required.</p>
		<a href="/create-instance" class="inline-block bg-indigo-600 text-white px-6 py-3 rounded-lg hover:bg-indigo-500 transition-colors font-semibold">Start Free Trial</a>
	</div>
</article>
`,

		"why-choose-ranx-cloud-for-n8n": `
<article class="prose prose-invert prose-lg max-w-none">
	<p class="lead text-xl text-gray-300 mb-8">Looking for the easiest way to deploy and manage n8n? Discover why ranx.cloud is the perfect solution for self-hosting n8n.</p>

	<h2>The Problem with Traditional Self-Hosting</h2>
	<p>Self-hosting n8n gives you control and saves money, but traditionally comes with challenges:</p>
	<ul>
		<li>Complex server setup and configuration</li>
		<li>SSL certificate management</li>
		<li>Regular updates and maintenance</li>
		<li>Security hardening</li>
		<li>Scaling and monitoring</li>
	</ul>
	<p>These tasks require DevOps expertise and take hours to set up correctly.</p>

	<h2>The ranx.cloud Solution</h2>
	<p>ranx.cloud handles all the technical complexity while giving you the benefits of self-hosting:</p>

	<h3>1. Deploy in Under 2 Minutes</h3>
	<p>No terminal commands, no configuration files, no complicated setup. Just:</p>
	<ol>
		<li>Sign in with Google</li>
		<li>Choose your subdomain</li>
		<li>Click deploy</li>
	</ol>
	<p>Your n8n instance is ready at yourname.ranx.cloud with automatic SSL.</p>

	<h3>2. Enterprise-Grade Infrastructure</h3>
	<p>Your n8n instance runs on reliable cloud infrastructure:</p>
	<ul>
		<li>99.9% uptime SLA</li>
		<li>Global CDN for fast access</li>
		<li>Automatic backups</li>
		<li>DDoS protection</li>
		<li>Enterprise security</li>
	</ul>

	<h3>3. Automatic Updates</h3>
	<p>Never fall behind on n8n releases. We automatically update your instance to the latest version, ensuring you have the newest features and security patches.</p>

	<h3>4. SSL Certificates Included</h3>
	<p>Every instance comes with automatic SSL certificate generation and renewal. Your workflows are always encrypted and secure.</p>

	<h3>5. Unlimited Executions</h3>
	<p>Unlike cloud-based solutions that charge per execution, you pay one flat rate of $9/month. Run as many workflows as you need without worrying about overage charges.</p>

	<h2>Cost Comparison</h2>
	<p>Let's compare costs for 50,000 workflow executions per month:</p>
	<ul>
		<li><strong>n8n Cloud:</strong> ~€150/month</li>
		<li><strong>Zapier:</strong> ~$250/month</li>
		<li><strong>DIY Self-Hosting:</strong> ~$30/month (plus 10+ hours of setup/maintenance)</li>
		<li><strong>ranx.cloud:</strong> $9/month (2-minute setup, zero maintenance)</li>
	</ul>

	<h2>Perfect for Teams</h2>
	<p>ranx.cloud is ideal for:</p>
	<ul>
		<li><strong>Startups</strong> - Cost-effective automation without execution limits</li>
		<li><strong>Agencies</strong> - Deploy separate instances for each client</li>
		<li><strong>Developers</strong> - Full control with zero DevOps hassle</li>
		<li><strong>Enterprises</strong> - Data privacy with managed infrastructure</li>
	</ul>

	<h2>What Our Users Say</h2>
	<blockquote>
		<p>"I switched from Zapier to n8n on ranx.cloud and I'm saving $200/month while running more complex workflows. Setup took literally 2 minutes."</p>
	</blockquote>

	<h2>Security & Compliance</h2>
	<p>Your data security is our priority:</p>
	<ul>
		<li>All data encrypted in transit and at rest</li>
		<li>Enterprise-grade secure infrastructure</li>
		<li>Regular security audits</li>
		<li>Isolated instances for each customer</li>
		<li>No access to your workflow data</li>
	</ul>

	<h2>Getting Started is Risk-Free</h2>
	<p>Try ranx.cloud with zero risk:</p>
	<ul>
		<li>✅ 3-day free trial</li>
		<li>✅ No credit card required</li>
		<li>✅ 7-day money-back guarantee</li>
		<li>✅ Cancel anytime</li>
	</ul>

	<h2>Support When You Need It</h2>
	<p>While we've made deployment simple, we're here to help:</p>
	<ul>
		<li>Email support at support@ranx.cloud</li>
		<li>Comprehensive documentation</li>
		<li>Active community</li>
	</ul>

	<div class="bg-gradient-to-r from-indigo-900/40 to-purple-900/40 border border-indigo-500/30 rounded-lg p-8 my-8">
		<h3 class="text-white mt-0 text-2xl">Ready to Deploy n8n?</h3>
		<p class="mb-6 text-lg">Join hundreds of teams automating their workflows on ranx.cloud. Start your free trial today.</p>
		<div class="flex gap-4 flex-wrap">
			<a href="/create-instance" class="inline-block bg-indigo-600 text-white px-8 py-4 rounded-lg hover:bg-indigo-500 transition-colors font-semibold text-lg">Start Free Trial</a>
			<a href="/pricing" class="inline-block bg-gray-700 text-white px-8 py-4 rounded-lg hover:bg-gray-600 transition-colors font-semibold text-lg">View Pricing</a>
		</div>
	</div>
</article>
`,
	}

	if html, ok := content[slug]; ok {
		return html
	}
	return ""
}
