import { Resource } from "sst";
import { SESv2Client, SendEmailCommand } from "@aws-sdk/client-sesv2";

const client = new SESv2Client();

export const handler = async () => {
	await client.send(
		new SendEmailCommand({
			FromEmailAddress: `no-reply@${Resource.MyEmail.sender}`,
			Destination: {
				ToAddresses: [`jhon@${Resource.MyEmail.sender}`],
			},
			Content: {
				Simple: {
					Subject: {
						Data: "Hello World!",
					},
					Body: {
						Text: {
							Data: "Sent from my SST app.",
						},
					},
				},
			},
		})
	);

	return {
		statusCode: 200,
		body: "Sent!"
	};
};
